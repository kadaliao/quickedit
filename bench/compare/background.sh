#!/bin/bash
# background.sh: every editor in the same state, running in the background
# without a window, then asked to open a file, as a double click does.
# Each editor first opens a file and has its window closed; QuickEdit is
# treated the same way. Results go to results/background.jsonl.
#
#   APPS_DIR=dir-with-downloaded-apps ./background.sh [runs]
set -u
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
runs=${1:-3}
apps_dir=${APPS_DIR:-$here/apps}
work=${WORK_DIR:-$here/work}/background
src=${WORK_DIR:-$here/work}
out=${OUT:-$here/results/background.jsonl}
measure=$here/measure
files=(small.go big.go doc.md huge.log)
vscode_data=${TMPDIR%/}/qe-vscode
apps=(
  "QuickEdit|${QE_APP:-/Applications/QuickEdit.app}|"
  "TextEdit|/System/Applications/TextEdit.app|"
  "Sublime Text|$apps_dir/Sublime Text.app|"
  "CotEditor|$apps_dir/CotEditor.app|"
  "Zed|$apps_dir/Zed.app|"
  "VS Code|$apps_dir/Visual Studio Code.app|--user-data-dir=$vscode_data/user --extensions-dir=$vscode_data/ext"
)
mkdir -p "$work" "$(dirname "$out")"
[ -f "$src/huge.log" ] || python3 -I "$here/genlog.py" "$src/huge.log"
for f in small.go big.go doc.md; do [ -f "$src/$f" ] || cp "$root/bench/files/$f" "$src/$f"; done

# Session restore off, so that a start opens only the file asked.
st="$HOME/Library/Application Support/Sublime Text/Packages/User"
mkdir -p "$st" "$HOME/.config/zed" "$vscode_data/user/User"
printf '{ "hot_exit": "disabled", "remember_open_files": false, "update_check": false }\n' > "$st/Preferences.sublime-settings"
printf '{ "restore_on_startup": "none", "auto_update": false, "telemetry": { "diagnostics": false, "metrics": false } }\n' > "$HOME/.config/zed/settings.json"
printf '{ "window.restoreWindows": "none", "workbench.startupEditor": "none", "update.mode": "none", "telemetry.telemetryLevel": "off", "security.workspace.trust.enabled": false, "workbench.tips.enabled": false }\n' > "$vscode_data/user/User/settings.json"
defaults write com.coteditor.CotEditor ApplePersistenceIgnoreState -bool YES

bundle_id() { /usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$1/Contents/Info.plist"; }

# close_windows clicks the close button of each window of the app, and
# prints how many windows are left.
close_windows() {
  osascript - "$1" <<'AS' 2>/dev/null
on run argv
  tell application "System Events"
    set p to first application process whose bundle identifier is (item 1 of argv)
    repeat 3 times
      if (count of windows of p) is 0 then exit repeat
      repeat with w in (get windows of p)
        try
          click (first button of w whose subrole is "AXCloseButton")
        end try
      end repeat
      delay 0.3
    end repeat
    return count of windows of p
  end tell
end run
AS
}

quit_app() {
  local path=$1 id; id=$(bundle_id "$path")
  osascript -e "tell application id \"$id\" to quit" >/dev/null 2>&1 &
  local q=$!
  for _ in $(seq 30); do pgrep -f "^$path/Contents/MacOS/" >/dev/null || break; sleep 0.1; done
  kill $q 2>/dev/null; pkill -9 -f "^$path/Contents/MacOS/" 2>/dev/null
  rm -rf "$HOME/Library/Saved Application State/$id.savedState" \
    "$HOME/Library/Containers/$id/Data/Library/Saved Application State/$id.savedState"
  sleep 0.5
}

old_anim=$(defaults read -g NSAutomaticWindowAnimationsEnabled 2>/dev/null)
defaults write -g NSAutomaticWindowAnimationsEnabled -bool false
dirs=("$HOME/Library/Application Support" "$HOME/Library/Caches" "$HOME/Library/Preferences" \
  "$HOME/Library/Saved Application State" "$HOME/Library/HTTPStorages" "$HOME/Library/Containers" \
  "$HOME/Library/Group Containers" "$HOME/Library/Application Scripts" "$HOME/Library/Logs" \
  "$HOME/Library/WebKit" "$HOME/.config")
before=$(for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done)
recents="$HOME/Library/Application Support/QuickEdit/recent-documents.json"
[ -f "$recents" ] && cp "$recents" "$work/recent-documents.json.bak"
cleanup() {
  if [ -z "$old_anim" ]; then defaults delete -g NSAutomaticWindowAnimationsEnabled 2>/dev/null
  else defaults write -g NSAutomaticWindowAnimationsEnabled -int "$old_anim"; fi
  pkill -x QuickEdit; sleep 1
  [ -f "$work/recent-documents.json.bak" ] && cp "$work/recent-documents.json.bak" "$recents"
  open -a "${QE_APP:-/Applications/QuickEdit.app}" --args --mygo-opened-at-login
  for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done |
    grep -vxF "$before" | grep -iE '/(sublime text|zed|code)$|com\.sublimetext|dev\.zed|com\.microsoft\.vscode|coteditor' |
    while IFS= read -r p; do rm -rf "$p"; done
}
trap cleanup EXIT

pkill -x QuickEdit; sleep 0.5
for entry in "${apps[@]}"; do
  IFS='|' read -r name path args <<< "$entry"
  [ -d "$path" ] || { echo "skip $name: no $path" >&2; continue; }
  id=$(bundle_id "$path")
  extra=(); for a in $args; do extra+=(--arg "$a"); done
  # Started with a file, then left without windows.
  d="$work/$name"; mkdir -p "$d"; cp "$src/small.go" "$d/start.go"
  "$measure" --app "$path" --file "$d/start.go" --no-memory --timeout 60 ${extra[@]+"${extra[@]}"} >/dev/null
  left=$(close_windows "$id"); sleep 1.5
  for run in $(seq "$runs"); do
    for f in "${files[@]}"; do
      cp "$src/$f" "$d/$run-$f"
      timeout=30; [[ $f == *huge* ]] && timeout=90
      # An editor that quit with its last window would start cold.
      running=false; pgrep -f "^$path/Contents/MacOS/" >/dev/null && running=true
      line=$("$measure" --app "$path" --file "$d/$run-$f" --warm --no-memory --timeout $timeout)
      echo "${line%\}},\"name\":\"$name\",\"run\":$run,\"running\":$running,\"windowsBefore\":${left:-0},\"file\":\"$f\"}" | tee -a "$out"
      left=$(close_windows "$id"); sleep 1
    done
  done
  quit_app "$path"
done

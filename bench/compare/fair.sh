#!/bin/bash
# fair.sh: every editor, one after the other in one session, opens the
# same files in the same three states, as a double click does:
#
#   cold        the editor is not running
#   background  it runs without a window: it opened a file, whose window
#               was closed
#   window      it shows a file: the new one opens as a tab, or a window
#
# Everyone shows the same thing: the text, highlighted. QuickEdit opens
# Markdown in its source for it (View > Open Markdown in Preview off), as
# the others do. Results go to results/fair.jsonl.
#
#   APPS_DIR=dir-with-downloaded-apps ./fair.sh [runs] [editor...]
set -u
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
runs=${1:-3}; shift || true
apps_dir=${APPS_DIR:-$here/apps}
src=${WORK_DIR:-$here/work}
work=$src/fair
out=${OUT:-$here/results/fair.jsonl}
shots=${SHOTS:-}
measure=$here/measure
qe_app=${QE_APP:-/Applications/QuickEdit.app}
files=(small.go big.go doc.md huge.log)
vscode_data=${TMPDIR%/}/qe-vscode
apps=(
  "QuickEdit|$qe_app|"
  "Sublime Text|$apps_dir/Sublime Text.app|"
  "Zed|$apps_dir/Zed.app|"
  "CotEditor|$apps_dir/CotEditor.app|"
  "VS Code|$apps_dir/Visual Studio Code.app|--user-data-dir=$vscode_data/user --extensions-dir=$vscode_data/ext"
)
mkdir -p "$work" "$(dirname "$out")"
[ -f "$src/huge.log" ] || python3 -I "$here/genlog.py" "$src/huge.log"
for f in small.go big.go doc.md; do cp "$root/bench/files/$f" "$src/$f"; done

# Session restore off, so that a start opens only the file asked.
st="$HOME/Library/Application Support/Sublime Text/Packages/User"
mkdir -p "$st" "$HOME/.config/zed" "$vscode_data/user/User"
printf '{ "hot_exit": "disabled", "remember_open_files": false, "update_check": false }\n' > "$st/Preferences.sublime-settings"
# Zed asks about every file outside a trusted project, over the dimmed
# editor, which then shows no highlighting: trusted, as after the user's
# first answer, it shows the file as usual.
printf '{ "restore_on_startup": "none", "auto_update": false, "session": { "trust_all_worktrees": true }, "enable_language_server": false, "telemetry": { "diagnostics": false, "metrics": false } }\n' > "$HOME/.config/zed/settings.json"
printf '{ "window.restoreWindows": "none", "workbench.startupEditor": "none", "update.mode": "none", "telemetry.telemetryLevel": "off", "security.workspace.trust.enabled": false, "workbench.tips.enabled": false }\n' > "$vscode_data/user/User/settings.json"
defaults write com.coteditor.CotEditor ApplePersistenceIgnoreState -bool YES

bundle_id() { /usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$1/Contents/Info.plist"; }
running() { pgrep -f "^$1/Contents/MacOS/" >/dev/null; }

# close_windows clicks the close button of each window of the app, and
# prints how many are left.
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
  for _ in $(seq 30); do running "$path" || break; sleep 0.1; done
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
qdata="$HOME/Library/Application Support/QuickEdit"
for f in recent-documents.json settings.json; do [ -f "$qdata/$f" ] && cp "$qdata/$f" "$work/$f.bak"; done
cleanup() {
  if [ -z "$old_anim" ]; then defaults delete -g NSAutomaticWindowAnimationsEnabled 2>/dev/null
  else defaults write -g NSAutomaticWindowAnimationsEnabled -int "$old_anim"; fi
  pkill -x QuickEdit; sleep 1
  for f in recent-documents.json settings.json; do [ -f "$work/$f.bak" ] && cp "$work/$f.bak" "$qdata/$f"; done
  open -a "$qe_app" --args --mygo-opened-at-login
  for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done |
    grep -vxF "$before" | grep -iE '/(sublime text|zed|code)$|com\.sublimetext|dev\.zed|com\.microsoft\.vscode|coteditor' |
    while IFS= read -r p; do rm -rf "$p"; done
}
trap cleanup EXIT
# QuickEdit shows Markdown's source, as the others do.
pkill -x QuickEdit; sleep 0.5
python3 -I - "$qdata/settings.json" <<'PY'
import json, sys
p = sys.argv[1]
try: s = json.load(open(p))
except Exception: s = {"fontSize": 13}
s["markdownSource"] = True
json.dump(s, open(p, "w"))
PY

# measure_one name path state file run [measure options]
measure_one() {
  local name=$1 path=$2 state=$3 f=$4 run=$5; shift 5
  local d="$work/$name/$state" timeout=30 opts=(--no-memory)
  mkdir -p "$d"; cp "$src/$f" "$d/$run-$f"
  [[ $f == huge.log ]] && timeout=90
  # The memory of each editor after its cold start with the big file.
  [ "$state" = cold ] && [ "$f" = big.go ] && opts=()
  local shot=()
  if [ -n "$shots" ] && [ "$run" = 1 ]; then
    mkdir -p "$shots/$name-$state-$f"; shot=(env "MEASURE_SHOTS=$shots/$name-$state-$f")
  fi
  local was=false; running "$path" && was=true
  local line
  line=$(${shot[@]+"${shot[@]}"} "$measure" --app "$path" --file "$d/$run-$f" --timeout $timeout ${opts[@]+"${opts[@]}"} "$@")
  echo "${line%\}},\"name\":\"$name\",\"state\":\"$state\",\"run\":$run,\"wasRunning\":$was,\"file\":\"$f\"}" | tee -a "$out"
}

want=("$@")
for entry in "${apps[@]}"; do
  IFS='|' read -r name path args <<< "$entry"
  if [ ${#want[@]} -gt 0 ] && ! printf '%s\n' "${want[@]}" | grep -qx "$name"; then continue; fi
  [ -d "$path" ] || { echo "skip $name: no $path" >&2; continue; }
  id=$(bundle_id "$path")
  extra=(); for a in $args; do extra+=(--arg "$a"); done
  start="$work/$name-start.go"; cp "$src/small.go" "$start"
  echo "== $name" >&2
  # The first start of an app is checked by the system: not measured.
  quit_app "$path"
  "$measure" --app "$path" --file "$start" --no-memory --timeout 60 ${extra[@]+"${extra[@]}"} >/dev/null
  quit_app "$path"

  for run in $(seq "$runs"); do
    for f in "${files[@]}"; do
      measure_one "$name" "$path" cold "$f" "$run" ${extra[@]+"${extra[@]}"}
      quit_app "$path"
    done
  done

  # Running without a window.
  "$measure" --app "$path" --file "$start" --no-memory --timeout 60 ${extra[@]+"${extra[@]}"} >/dev/null
  close_windows "$id" >/dev/null; sleep 1.5
  for run in $(seq "$runs"); do
    for f in "${files[@]}"; do
      measure_one "$name" "$path" background "$f" "$run" --warm
      close_windows "$id" >/dev/null; sleep 1
    done
  done
  quit_app "$path"

  # Showing a file.
  for run in $(seq "$runs"); do
    "$measure" --app "$path" --file "$start" --no-memory --timeout 60 ${extra[@]+"${extra[@]}"} >/dev/null; sleep 1
    for f in "${files[@]}"; do
      measure_one "$name" "$path" window "$f" "$run" --warm
      sleep 0.5
    done
    quit_app "$path"
  done
done

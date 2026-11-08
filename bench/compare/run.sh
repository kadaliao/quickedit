#!/bin/bash
# run.sh: opens the same files in QuickEdit and other editors, cold (the
# app not running) and warm (the app running), and records how long until
# each file shows. See README.md.
#
#   APPS_DIR=dir-with-downloaded-apps ./run.sh [runs] [app...]
#
# Results go to results/raw.jsonl, one line per measurement.
set -u
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
runs=${1:-5}; shift || true
apps_dir=${APPS_DIR:-$here/apps}
work=${WORK_DIR:-$here/work}
out=${OUT:-$here/results/raw.jsonl}
measure=$here/measure
mkdir -p "$here/results" "$work"

# name|app path|files|arguments at launch
# VS Code listens on a socket in its data folder, whose path must stay
# under 104 characters.
vscode_data=${TMPDIR%/}/qe-vscode
all_apps=(
  "QuickEdit|$root/build/darwin-arm64/QuickEdit.app|small.go big.go doc.md huge.log|"
  "TextEdit|/System/Applications/TextEdit.app|small.go big.go doc.md huge.log|"
  "Sublime Text|$apps_dir/Sublime Text.app|small.go big.go doc.md huge.log|"
  "CotEditor|$apps_dir/CotEditor.app|small.go big.go doc.md huge.log|"
  "Zed|$apps_dir/Zed.app|small.go big.go doc.md huge.log|"
  "VS Code|$apps_dir/Visual Studio Code.app|small.go big.go doc.md huge.log|--user-data-dir=$vscode_data/user --extensions-dir=$vscode_data/ext"
  "Typora|/Applications/Typora.app|doc.md|"
)

# The test files: bench/files, and a 44 MB log; w-* are the copies opened warm.
prepare_files() {
  for f in small.go big.go doc.md; do cp "$root/bench/files/$f" "$work/$f"; done
  [ -f "$work/huge.log" ] || python3 -I "$here/genlog.py" "$work/huge.log"
  for f in small.go big.go doc.md huge.log; do cp "$work/$f" "$work/w-$f" && cp "$work/$f" "$work/start-$f"; done
}

# Session restore off, so that each cold start opens the one file asked.
prepare_settings() {
  local st="$HOME/Library/Application Support/Sublime Text/Packages/User"
  mkdir -p "$st"
  printf '{ "hot_exit": "disabled", "remember_open_files": false, "update_check": false }\n' > "$st/Preferences.sublime-settings"
  mkdir -p "$HOME/.config/zed"
  printf '{ "restore_on_startup": "none", "auto_update": false, "telemetry": { "diagnostics": false, "metrics": false } }\n' > "$HOME/.config/zed/settings.json"
  mkdir -p "$vscode_data/user/User"
  printf '{ "window.restoreWindows": "none", "workbench.startupEditor": "none", "update.mode": "none", "telemetry.telemetryLevel": "off", "security.workspace.trust.enabled": false, "workbench.tips.enabled": false }\n' > "$vscode_data/user/User/settings.json"
  # CotEditor restores every window it ever showed, even with its saved
  # state removed, and when killed: ignore it, as AppKit lets apps do.
  defaults write com.coteditor.CotEditor ApplePersistenceIgnoreState -bool YES
}

bundle_id() { /usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$1/Contents/Info.plist"; }

# quit closes the app's windows and quits it, killing it if it hangs.
quit_app() {
  local path=$1 id; id=$(bundle_id "$path")
  osascript -e "tell application id \"$id\" to quit" >/dev/null 2>&1 &
  local q=$!
  for _ in $(seq 50); do
    pgrep -f "^$path/Contents/MacOS/" >/dev/null || break
    sleep 0.1
  done
  kill $q 2>/dev/null
  pkill -9 -f "^$path/Contents/MacOS/" 2>/dev/null
  # Forget the windows, so that the next start restores none.
  rm -rf "$HOME/Library/Saved Application State/$id.savedState" \
    "$HOME/Library/Containers/$id/Data/Library/Saved Application State/$id.savedState"
  sleep 1
}

measure_one() { # name path args file mode
  local name=$1 path=$2 args=$3 file=$4 mode=$5 run=$6 extra=() a
  for a in $args; do extra+=(--arg "$a"); done
  local timeout=30
  [[ $file == *huge* ]] && timeout=90
  local line
  if [ "$mode" = warm ]; then
    line=$("$measure" --app "$path" --file "$work/w-$file" --warm --timeout $timeout)
  else
    line=$("$measure" --app "$path" --file "$work/$file" --timeout $timeout ${extra[@]+"${extra[@]}"})
  fi
  echo "${line%\}},\"name\":\"$name\",\"run\":$run}" | tee -a "$out"
}

old_anim=$(defaults read -g NSAutomaticWindowAnimationsEnabled 2>/dev/null)
defaults write -g NSAutomaticWindowAnimationsEnabled -bool false
restore() {
  if [ -z "$old_anim" ]; then defaults delete -g NSAutomaticWindowAnimationsEnabled 2>/dev/null
  else defaults write -g NSAutomaticWindowAnimationsEnabled -int "$old_anim"; fi
}
# What the editors write in the home directory, listed before they run, so
# that only what they add is removed afterwards.
dirs=("$HOME/Library/Application Support" "$HOME/Library/Caches" "$HOME/Library/Preferences" \
  "$HOME/Library/Saved Application State" "$HOME/Library/HTTPStorages" "$HOME/Library/Containers" \
  "$HOME/Library/Group Containers" "$HOME/Library/Application Scripts" "$HOME/Library/Logs" \
  "$HOME/Library/WebKit" "$HOME/.config" \
  "$HOME/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments")
before=$(for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done)
# QuickEdit keeps the files it opened in its list of recent documents, which
# the user's own QuickEdit shares: put the list back afterwards.
recents="$HOME/Library/Application Support/QuickEdit/recent-documents.json"
[ -f "$recents" ] && cp "$recents" "$work/recent-documents.json.bak"
cleanup() {
  restore
  [ -f "$work/recent-documents.json.bak" ] && cp "$work/recent-documents.json.bak" "$recents"
  [ -n "${KEEP_EDITOR_DATA:-}" ] && return
  for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done |
    grep -vxF "$before" | grep -iE '/(sublime text|zed|code)$|com\.sublimetext|dev\.zed|com\.microsoft\.vscode|coteditor' |
    while IFS= read -r p; do echo "cleanup: $p" >&2; rm -rf "$p"; done
}
trap cleanup EXIT
prepare_files
prepare_settings

want=("$@")
for entry in "${all_apps[@]}"; do
  IFS='|' read -r name path files args <<< "$entry"
  if [ ${#want[@]} -gt 0 ] && ! printf '%s\n' "${want[@]}" | grep -qx "$name"; then continue; fi
  [ -d "$path" ] || { echo "skip $name: no $path" >&2; continue; }
  if pgrep -f "^$path/Contents/MacOS/" >/dev/null; then echo "skip $name: it is running" >&2; continue; fi
  echo "== $name" >&2
  start=(--app "$path" --file "$work/start-${files%% *}" --timeout 90)
  for a in $args; do start+=(--arg "$a"); done
  # The first start of an app is checked by the system: not measured.
  "$measure" "${start[@]}" >/dev/null
  quit_app "$path"
  for run in $(seq "$runs"); do
    for f in $files; do
      measure_one "$name" "$path" "$args" "$f" cold "$run"
      quit_app "$path"
    done
    "$measure" "${start[@]}" >/dev/null
    for f in $files; do
      measure_one "$name" "$path" "$args" "$f" warm "$run"
    done
    quit_app "$path"
  done
done

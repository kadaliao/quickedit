#!/bin/bash
# project.sh: opens real files of this repository in each editor, as a
# double click does when no editor window is open: QuickEdit waiting in the
# background (as after login), the others cold. One run per file, no
# memory reading, to fit in a few minutes. See README.md.
#
#   APPS_DIR=dir-with-downloaded-apps ./project.sh
set -u
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
apps_dir=${APPS_DIR:-$here/apps}
work=${WORK_DIR:-$here/work}/project
out=${OUT:-$here/results/project.jsonl}
measure=$here/measure
qe_app=${QE_APP:-/Applications/QuickEdit.app}
files=(view.go README.md bench/compare/results/summary.json bench/compare/page/index.html Makefile)
vscode_data=${TMPDIR%/}/qe-vscode
apps=(
  "TextEdit|/System/Applications/TextEdit.app|"
  "Sublime Text|$apps_dir/Sublime Text.app|"
  "CotEditor|$apps_dir/CotEditor.app|"
  "Zed|$apps_dir/Zed.app|"
  "VS Code|$apps_dir/Visual Studio Code.app|--user-data-dir=$vscode_data/user --extensions-dir=$vscode_data/ext"
)
mkdir -p "$work" "$(dirname "$out")"

# Session restore off, so that each cold start opens the one file asked.
st="$HOME/Library/Application Support/Sublime Text/Packages/User"
mkdir -p "$st" "$HOME/.config/zed" "$vscode_data/user/User"
printf '{ "hot_exit": "disabled", "remember_open_files": false, "update_check": false }\n' > "$st/Preferences.sublime-settings"
printf '{ "restore_on_startup": "none", "auto_update": false, "telemetry": { "diagnostics": false, "metrics": false } }\n' > "$HOME/.config/zed/settings.json"
printf '{ "window.restoreWindows": "none", "workbench.startupEditor": "none", "update.mode": "none", "telemetry.telemetryLevel": "off", "security.workspace.trust.enabled": false, "workbench.tips.enabled": false }\n' > "$vscode_data/user/User/settings.json"
defaults write com.coteditor.CotEditor ApplePersistenceIgnoreState -bool YES

quit_app() {
  local path=$1 id; id=$(/usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$path/Contents/Info.plist")
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
  open -a "$qe_app" --args --mygo-opened-at-login
  for d in "${dirs[@]}"; do ls -1 "$d" 2>/dev/null | sed "s|^|$d/|"; done |
    grep -vxF "$before" | grep -iE '/(sublime text|zed|code)$|com\.sublimetext|dev\.zed|com\.microsoft\.vscode|coteditor' |
    while IFS= read -r p; do rm -rf "$p"; done
}
trap cleanup EXIT

# A fresh copy of each file per editor, so that none is open already.
copy() { mkdir -p "$work/$1"; cp "$root/$2" "$work/$1/$(basename "$2")"; echo "$work/$1/$(basename "$2")"; }
emit() { echo "${2%\}},\"name\":\"$1\"}" | tee -a "$out"; }

# QuickEdit, waiting in the background as after login.
for f in "${files[@]}"; do
  pkill -x QuickEdit; sleep 0.5
  open -a "$qe_app" --args --mygo-opened-at-login; sleep 2.5
  emit "QuickEdit 常驻" "$("$measure" --app "$qe_app" --file "$(copy qe "$f")" --warm --no-memory)"
done
pkill -x QuickEdit

for entry in "${apps[@]}"; do
  IFS='|' read -r name path args <<< "$entry"
  [ -d "$path" ] || { echo "skip $name: no $path" >&2; continue; }
  extra=(); for a in $args; do extra+=(--arg "$a"); done
  # The first start of an app is checked by the system: not measured.
  "$measure" --app "$path" --file "$(copy "start-$name" README.md)" --no-memory --timeout 60 ${extra[@]+"${extra[@]}"} >/dev/null
  quit_app "$path"
  for f in "${files[@]}"; do
    emit "$name" "$("$measure" --app "$path" --file "$(copy "$name" "$f")" --no-memory ${extra[@]+"${extra[@]}"})"
    quit_app "$path"
  done
done

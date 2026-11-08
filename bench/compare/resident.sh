#!/bin/bash
# resident.sh [runs]: QuickEdit as it runs after login, waiting in the
# background without a window, then asked to open each file: what a double
# click does on a Mac where QuickEdit starts at login. Appends to
# results/raw.jsonl as editor "QuickEdit 常驻", mode "cold", since for the
# user no editor window is open, as in a cold start.
set -u
here=$(cd "$(dirname "$0")" && pwd)
runs=${1:-5}
app=${APP:-/Applications/QuickEdit.app}
work=${WORK_DIR:-$here/work}
out=${OUT:-$here/results/raw.jsonl}
recents="$HOME/Library/Application Support/QuickEdit/recent-documents.json"
[ -f "$recents" ] && cp "$recents" "$work/recent-documents.json.bak"
old_anim=$(defaults read -g NSAutomaticWindowAnimationsEnabled 2>/dev/null)
defaults write -g NSAutomaticWindowAnimationsEnabled -bool false
finish() {
  if [ -z "$old_anim" ]; then defaults delete -g NSAutomaticWindowAnimationsEnabled 2>/dev/null
  else defaults write -g NSAutomaticWindowAnimationsEnabled -int "$old_anim"; fi
  pkill -x QuickEdit; sleep 1
  [ -f "$work/recent-documents.json.bak" ] && cp "$work/recent-documents.json.bak" "$recents"
  # Back to waiting in the background, as after login.
  open -a "$app" --args --mygo-opened-at-login
}
trap finish EXIT
for run in $(seq "$runs"); do
  for f in small.go big.go doc.md huge.log; do
    pkill -x QuickEdit; sleep 1
    open -a "$app" --args --mygo-opened-at-login; sleep 3
    cp "$work/$f" "$work/r-$f"
    timeout=30; [[ $f == *huge* ]] && timeout=90
    line=$("$here/measure" --app "$app" --file "$work/r-$f" --warm --timeout $timeout)
    echo "${line%\}},\"name\":\"QuickEdit 常驻\",\"run\":$run}" | sed 's/"mode":"warm"/"mode":"cold"/; s/"file":"r-/"file":"/' | tee -a "$out"
  done
done

#!/bin/sh
# Builds build/QE-mygo.app and build/QE-native.app for `run -resident`.
# The document type's LSHandlerRank is None, so they never become the default app.
set -e
cd "$(dirname "$0")/.."
mkdir -p build
go build -trimpath -tags mygo_noinspector -ldflags "-s -w -X github.com/egoist/mygo.production=1" -o build/resident-mygo ./resident/mygo
swiftc -O resident/native/main.swift -o build/resident-native
for name in mygo native; do
  app=build/QE-$name.app
  rm -rf "$app"; mkdir -p "$app/Contents/MacOS"
  cp "build/resident-$name" "$app/Contents/MacOS/qe-$name"
  cat > "$app/Contents/Info.plist" <<PL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>dev.quickedit.bench.$name</string>
<key>CFBundleName</key><string>QE-$name</string>
<key>CFBundleExecutable</key><string>qe-$name</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>0.0.1</string>
<key>LSMinimumSystemVersion</key><string>13.0</string>
<key>NSHighResolutionCapable</key><true/>
<key>CFBundleDocumentTypes</key><array><dict>
  <key>CFBundleTypeName</key><string>Text</string>
  <key>CFBundleTypeRole</key><string>Editor</string>
  <key>LSHandlerRank</key><string>None</string>
  <key>LSItemContentTypes</key><array><string>public.data</string></array>
</dict></array>
</dict></plist>
PL
  codesign -s - --force "$app"
done

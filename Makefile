# Builds QuickEdit, with MyGo patched by patches/mygo (see scripts/patch-mygo.sh).
APP := build/darwin-arm64/QuickEdit.app

.PHONY: build dmg test run icon install clean mygo

# MyGo with QuickEdit's patches, which go.mod replaces the module with.
mygo:
	@scripts/patch-mygo.sh

build: mygo
	go tool mygo build -skip-dmg

dmg: mygo
	go tool mygo build

test: mygo
	go vet . && go test .

# make run FILE=README.md
run: mygo
	go run . $(FILE)

icon:
	swift scripts/make-icon.swift resources/icon.png

# Copies the app to /Applications and puts the qe command in ~/.local/bin.
install: build
	rm -rf /Applications/QuickEdit.app
	cp -R $(APP) /Applications/
	mkdir -p $(HOME)/.local/bin
	install -m 755 scripts/qe $(HOME)/.local/bin/qe

clean:
	rm -rf build .mygo

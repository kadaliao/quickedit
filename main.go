// QuickEdit is a fast editor and viewer for macOS: it opens any file at
// once, highlights code, renders Markdown, and stays out of the way.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// textExtensions are the files the documents claim by extension. Any
// other file opens too (see openAnyFile); these only make the system's
// Save panel and the documents' own routing know them.
var textExtensions = strings.Fields(`txt md markdown mdown mkd mdx log csv tsv
	json jsonc json5 yaml yml toml ini cfg conf env properties xml plist svg
	html htm css scss sass less js mjs cjs jsx ts tsx vue svelte astro
	go mod sum rs py pyi rb php java kt kts scala swift m mm h hh hpp c cc cpp cxx cs
	dart lua pl pm r sql sh bash zsh fish ps1 bat cmd mk cmake gradle proto
	graphql gql zig nim ex exs erl hs ml fs clj tf hcl diff patch srt vtt tex bib
	rst adoc org gitignore gitattributes editorconfig dockerfile lock`)

var docs *mygo.Documents[content]

func main() {
	mygo.App.SetLanguages("en", "zh-Hans")
	loadSettings()

	docs = mygo.NewDocuments(mygo.DocumentsOptions[content]{
		Types:  []mygo.DocumentType{{Name: "Text", Extensions: textExtensions}},
		Read:   readContent,
		Write:  writeContent,
		Equal:  equalContent,
		Window: newDocWindow,
		// QuickEdit opens files of any name, which the documents leave to
		// openAnyFile, so it makes the untitled document itself.
		NoUntitled: true,
		// A document in a tab not shown, before asking to save it.
		Focus: focusDocument,
	})
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) { quitting = !e.DefaultPrevented() })

	mygo.App.OnOpenFile(func(path string) {
		requested.Store(true)
		if !claimed(path) {
			mygo.App.WhenReady(func() { go openAnyFile(path) })
		}
	})
	// Files on the command line, as QuickEdit.app/Contents/MacOS/QuickEdit
	// notes.md: MyGo delivers only those of mygo.json's fileAssociations,
	// which lists none, as QuickEdit opens files of any name.
	for _, a := range os.Args[1:] {
		if p, ok := fileArg(a); ok {
			requested.Store(true)
			mygo.App.WhenReady(func() { go openAnyFile(p) })
		}
	}
	// The Dock icon of an app without windows makes a new document.
	mygo.App.OnActivate(func(visible bool) {
		if !visible && len(docs.Documents()) == 0 && pending.Load() == 0 {
			go docs.New()
		}
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(buildMenu())
		listen()
		// The files of the command line arrive once this returns, through
		// the main thread's queue: look after them. Started with nothing to
		// open and nothing to recover, show a new document, as Sublime Text
		// shows an empty one.
		//
		// Started at login (the App menu's choice), it waits in the
		// background, without a window, for the files the user opens.
		go func() {
			mygo.RunOnMain(func() {})
			mygo.RunOnMain(func() {
				if !requested.Load() && !mygo.App.WasOpenedAtLogin() && !hasRecoveryCopies() {
					go docs.New()
				}
				time.AfterFunc(500*time.Millisecond, func() { mygo.RunOnMain(prepareSpare) })
				time.AfterFunc(1500*time.Millisecond, func() { mygo.RunOnMain(warmText) })
			})
		}()
		undoLoginItemOffer()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// requested tells that the system asked the app to open a file.
var requested atomicBool

type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (b *atomicBool) Store(v bool) { b.mu.Lock(); b.v = v; b.mu.Unlock() }
func (b *atomicBool) Load() bool   { b.mu.Lock(); defer b.mu.Unlock(); return b.v }

// claimed reports whether the documents open path by its extension.
func claimed(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	for _, e := range textExtensions {
		if e == ext {
			return true
		}
	}
	return false
}

// fileArg returns the absolute path of a command-line argument naming a
// file or a folder.
func fileArg(a string) (string, bool) {
	if strings.HasPrefix(a, "-") {
		return "", false
	}
	p, err := filepath.Abs(a)
	if err != nil {
		return "", false
	}
	if info, err := os.Stat(p); err != nil || !info.Mode().IsRegular() && !info.IsDir() {
		return "", false
	}
	return p, true
}

// openAnyFile opens a file the documents don't claim by its extension,
// as Makefile or LICENSE, or a folder, in a window of its own.
func openAnyFile(path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		mygo.RunOnMain(func() { openFolder(filepath.Clean(path)) })
		return
	}
	if _, err := docs.Open(path); err != nil {
		log.Printf("open %s: %v", path, err)
	}
}

// openDialog asks for files of any kind, as the documents' own Open asks
// only for the kinds they claim. Folders open with openFolderDialog.
func openDialog() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.FocusedWindow(), Multiple: true})
	if err != nil {
		return
	}
	for _, p := range paths {
		go openAnyFile(p)
	}
}

// hasRecoveryCopies reports whether a crash or a quit left documents with
// unsaved changes, which the documents open as the app starts.
// Each copy is a file of a session directory, Recovery/<kind>/<session>/,
// beside the session's lock and index.json.
func hasRecoveryCopies() bool {
	found := false
	root := filepath.Join(userData(), "Recovery")
	filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() != "lock" && d.Name() != "index.json" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// tr returns the Chinese or the English text, by the language the app
// shows.
func tr(zh, en string) string {
	if strings.HasPrefix(mygo.App.Locale(), "zh") {
		return zh
	}
	return en
}

// appBindings take from the editor the keys of QuickEdit's menu items.
var appBindings = []editor.Binding{
	{Mods: ui.Cmd, Key: ui.KeyE, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Alt, Key: ui.KeyE, Command: editor.NoCommand},
	{Mods: ui.Alt, Key: ui.KeyZ, Command: editor.NoCommand},
	{Mods: ui.Cmd, Key: ui.KeyEqual, Command: editor.NoCommand},
	{Mods: ui.Cmd, Key: ui.KeyMinus, Command: editor.NoCommand},
	{Mods: ui.Cmd, Key: ui.Key0, Command: editor.NoCommand},
	{Mods: ui.Cmd, Key: ui.KeyW, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyW, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyBracketLeft, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyBracketRight, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyO, Command: editor.NoCommand},
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyB, Command: editor.NoCommand},
}

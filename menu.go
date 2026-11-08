package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// settings are the app's preferences, kept across launches.
type appSettings struct {
	fontSize float32
}

var settings = appSettings{fontSize: defaultFontSize}

const defaultFontSize = 13

type savedSettings struct {
	FontSize float32 `json:"fontSize"`
}

// userData is the app's directory of its own data.
func userData() string {
	dir, _ := mygo.App.Path(mygo.PathUserData)
	return dir
}

func settingsPath() string {
	return filepath.Join(userData(), "settings.json")
}

func loadSettings() {
	b, err := os.ReadFile(settingsPath())
	if err != nil {
		return
	}
	var s savedSettings
	if json.Unmarshal(b, &s) == nil && s.FontSize >= 8 && s.FontSize <= 40 {
		settings.fontSize = s.FontSize
	}
}

func saveSettings() {
	b, _ := json.MarshalIndent(savedSettings{FontSize: settings.fontSize}, "", "  ")
	os.MkdirAll(filepath.Dir(settingsPath()), 0o755)
	os.WriteFile(settingsPath(), b, 0o644)
}

// wrap tells whether lines wrap by default: prose does, as in TextEdit,
// and code doesn't, as in Sublime Text.
func (s *appSettings) wrap(prose bool) bool { return prose }

func setFontSize(size float32) {
	settings.fontSize = max(8, min(40, size))
	saveSettings()
	for w, v := range views {
		v.applySettings()
		w.Update(func() {})
	}
}

// onView runs fn with the focused window's view, if it shows a document.
func onView(fn func(v *docView)) func(*mygo.MenuItem, *mygo.Window) {
	return func(_ *mygo.MenuItem, w *mygo.Window) {
		if w == nil {
			w = mygo.FocusedWindow()
		}
		if v := viewOf(w); v != nil {
			fn(v)
			w.Update(func() {})
		}
	}
}

func buildMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: tr("文件", "File"), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleNew},
			{Label: tr("打开…", "Open…"), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { openDialog() }},
			{Role: mygo.RoleOpenRecent},
			mygo.Separator(),
			{Role: mygo.RoleClose},
			{Role: mygo.RoleSave},
			{Role: mygo.RoleDuplicate},
			{Role: mygo.RoleSaveAs},
			{Role: mygo.RoleRename},
			{Role: mygo.RoleMoveTo},
			{Role: mygo.RoleRevert},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: tr("显示", "View"), Submenu: []*mygo.MenuItem{
			{Label: tr("切换预览 / 源码", "Toggle Preview / Source"), Accelerator: "CmdOrCtrl+E",
				Click: onView(func(v *docView) { v.toggleMode() })},
			{Label: tr("并排预览", "Side-by-Side Preview"), Accelerator: "CmdOrCtrl+Alt+E",
				Click: onView(func(v *docView) {
					if v.mode == modeSplit {
						v.setMode(modePreview)
					} else {
						v.setMode(modeSplit)
					}
				})},
			mygo.Separator(),
			{Label: tr("自动换行", "Word Wrap"), Accelerator: "Alt+Z",
				Click: onView(func(v *docView) { v.st.SoftWrap = !v.st.SoftWrap })},
			mygo.Separator(),
			{Label: tr("放大", "Bigger"), Accelerator: "CmdOrCtrl+=", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(settings.fontSize + 1) }},
			{Label: tr("缩小", "Smaller"), Accelerator: "CmdOrCtrl+-", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(settings.fontSize - 1) }},
			{Label: tr("实际大小", "Actual Size"), Accelerator: "CmdOrCtrl+0", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(defaultFontSize) }},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

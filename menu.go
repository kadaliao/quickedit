package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
)

// settings are the app's preferences, kept across launches.
type appSettings struct {
	fontSize float32
	// loginItemOffered tells that QuickEdit 0.1 or 0.2 made itself start
	// at login, unasked, or the user chose it in the App menu then.
	loginItemOffered bool
	// loginItemChosen tells that the user chose in the App menu whether
	// QuickEdit starts at login.
	loginItemChosen bool
	// markdownSource opens Markdown files in their source rather than
	// their preview.
	markdownSource bool
}

var settings = appSettings{fontSize: defaultFontSize}

const defaultFontSize = 13

type savedSettings struct {
	FontSize         float32 `json:"fontSize"`
	LoginItemOffered bool    `json:"loginItemOffered,omitempty"`
	LoginItemChosen  bool    `json:"loginItemChosen,omitempty"`
	MarkdownSource   bool    `json:"markdownSource,omitempty"`
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
	if json.Unmarshal(b, &s) != nil {
		return
	}
	if s.FontSize >= 8 && s.FontSize <= 40 {
		settings.fontSize = s.FontSize
	}
	settings.loginItemOffered = s.LoginItemOffered
	settings.loginItemChosen = s.LoginItemChosen
	settings.markdownSource = s.MarkdownSource
}

func saveSettings() {
	b, _ := json.MarshalIndent(savedSettings{FontSize: settings.fontSize, LoginItemOffered: settings.loginItemOffered, LoginItemChosen: settings.loginItemChosen, MarkdownSource: settings.markdownSource}, "", "  ")
	os.MkdirAll(filepath.Dir(settingsPath()), 0o755)
	os.WriteFile(settingsPath(), b, 0o644)
}

// wrap tells whether lines wrap by default: prose does, as in TextEdit,
// and code doesn't, as in Sublime Text. Nor does a long file, as a log,
// which wrapping takes tens of milliseconds to show.
func (s *appSettings) wrap(prose bool, lines int) bool { return prose && lines <= wrapLines }

// wrapLines is the most lines a file wrapped by default has.
const wrapLines = 20_000

func setFontSize(size float32) {
	settings.fontSize = max(8, min(40, size))
	saveSettings()
	for w, h := range hosts {
		for _, v := range h.tabs {
			v.applySettings()
		}
		w.Update(func() {})
	}
}

func orFocused(w *mygo.Window) *mygo.Window {
	if w == nil {
		return mygo.FocusedWindow()
	}
	return w
}

// onWindow runs fn with the focused document window, if any.
func onWindow(fn func(w *docWindow)) func(*mygo.MenuItem, *mygo.Window) {
	return func(_ *mygo.MenuItem, w *mygo.Window) {
		if h := hosts[orFocused(w)]; h != nil && len(h.tabs) > 0 {
			fn(h)
		}
	}
}

// onView runs fn with the focused window's view, if it shows a document.
func onView(fn func(v *docView)) func(*mygo.MenuItem, *mygo.Window) {
	return func(_ *mygo.MenuItem, w *mygo.Window) {
		w = orFocused(w)
		if v := viewOf(w); v != nil {
			fn(v)
			w.Update(func() {})
		}
	}
}

// installed reports whether this is the app in /Applications or
// ~/Applications, as opposed to a build being tried out or measured.
func installed() bool {
	exe, err := os.Executable()
	if err != nil || !mygo.App.IsPackaged() {
		return false
	}
	home, _ := os.UserHomeDir()
	return strings.HasPrefix(exe, "/Applications/") || strings.HasPrefix(exe, filepath.Join(home, "Applications")+"/")
}

// undoLoginItemOffer takes back, once, the start at login that QuickEdit
// 0.1 and 0.2 chose for the user: it starts at login only when the user
// chooses so in the App menu.
func undoLoginItemOffer() {
	if !settings.loginItemOffered || settings.loginItemChosen {
		return
	}
	if mygo.App.OpenAtLogin() {
		if err := mygo.App.SetOpenAtLogin(false); err != nil {
			log.Printf("open at login: %v", err)
			return
		}
	}
	settings.loginItemOffered = false
	saveSettings()
	if item := mygo.App.Menu().ItemByID("open-at-login"); item != nil {
		item.SetChecked(false)
	}
}

func toggleOpenAtLogin(item *mygo.MenuItem, _ *mygo.Window) {
	// The item is checked or unchecked already, as it was chosen.
	if err := mygo.App.SetOpenAtLogin(item.Checked); err != nil {
		log.Printf("open at login: %v", err)
		item.SetChecked(mygo.App.OpenAtLogin())
		return
	}
	settings.loginItemChosen = true
	saveSettings()
}

func buildMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "QuickEdit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			mygo.Separator(),
			{ID: "open-at-login", Label: tr("登录时在后台启动", "Open at Login in the Background"), Type: mygo.MenuItemCheckbox,
				Checked: installed() && mygo.App.OpenAtLogin(), Disabled: !installed(), Click: toggleOpenAtLogin},
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}},
		{Label: tr("文件", "File"), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleNew},
			{Label: tr("打开…", "Open…"), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { openDialog() }},
			{Role: mygo.RoleOpenRecent},
			mygo.Separator(),
			{Label: tr("关闭标签页", "Close Tab"), Accelerator: "CmdOrCtrl+W", Click: onWindow(func(w *docWindow) { w.closeTab(w.current(), nil) })},
			{Label: tr("关闭窗口", "Close Window"), Accelerator: "CmdOrCtrl+Shift+W", Click: func(_ *mygo.MenuItem, w *mygo.Window) {
				if w = orFocused(w); w != nil {
					w.Close()
				}
			}},
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
			{Label: tr("打开 Markdown 时显示预览", "Open Markdown in Preview"), Type: mygo.MenuItemCheckbox, Checked: !settings.markdownSource,
				Click: func(item *mygo.MenuItem, _ *mygo.Window) {
					settings.markdownSource = !settings.markdownSource
					item.SetChecked(!settings.markdownSource)
					saveSettings()
				}},
			mygo.Separator(),
			{Label: tr("自动换行", "Word Wrap"), Accelerator: "Alt+Z",
				Click: onView(func(v *docView) { v.st.SoftWrap = !v.st.SoftWrap })},
			mygo.Separator(),
			{Label: tr("放大", "Bigger"), Accelerator: "CmdOrCtrl+=", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(settings.fontSize + 1) }},
			{Label: tr("缩小", "Smaller"), Accelerator: "CmdOrCtrl+-", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(settings.fontSize - 1) }},
			{Label: tr("实际大小", "Actual Size"), Accelerator: "CmdOrCtrl+0", Click: func(*mygo.MenuItem, *mygo.Window) { setFontSize(defaultFontSize) }},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
			mygo.Separator(),
			{Label: tr("显示上一个标签页", "Show Previous Tab"), Accelerator: "CmdOrCtrl+Shift+[", Click: onWindow(func(w *docWindow) { w.next(-1) })},
			{Label: tr("显示下一个标签页", "Show Next Tab"), Accelerator: "CmdOrCtrl+Shift+]", Click: onWindow(func(w *docWindow) { w.next(1) })},
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

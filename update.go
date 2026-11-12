package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/netproxy"
)

// QuickEdit asks GitHub for its newest release once a day, and tells the
// user of a newer one once, with a dialog they can put off or skip. It
// installs nothing itself: the dialog opens the release's page, or, for
// the app that Homebrew installed, gives the command that updates it.
// Homebrew's cask follows a release within hours (homebrew-tap's
// bump.yml), and brew upgrade installs only what the cask has, so the
// app from Homebrew is told of a release once the cask has it.

// The newest release, and the cask Homebrew installs the app with.
var (
	releaseAPI = "https://api.github.com/repos/kadaliao/quickedit/releases/latest"
	caskAPI    = "https://api.github.com/repos/kadaliao/homebrew-tap/contents/Casks/quickedit.rb"
)

// brewUpgrade updates the app that Homebrew installed. brew update first,
// as brew upgrade may read a tap fetched a day ago.
const brewUpgrade = "brew update && brew upgrade --cask quickedit"

// updateInterval is how often the app asks for a newer release.
const updateInterval = 24 * time.Hour

// release is a release of QuickEdit on GitHub.
type release struct {
	// Version is the release's tag without its v.
	Version string
	// URL is the release's page, with its notes and its DMG.
	URL string
	// Highlights are the titles of the notes' points: "- **文件树**：…".
	Highlights []string
}

// updateCheck is what a check found.
type updateCheck struct {
	current string
	latest  release
	// brew tells that Homebrew installed the app, and cask is the version
	// its cask installs.
	brew bool
	cask string
}

// offer returns the release the user may update to, or false when the
// app is the newest that it can update to.
func (c updateCheck) offer() (release, bool) {
	if !c.brew {
		if !newerVersion(c.latest.Version, c.current) {
			return release{}, false
		}
		return c.latest, true
	}
	if !newerVersion(c.cask, c.current) {
		return release{}, false
	}
	if c.cask == c.latest.Version {
		return c.latest, true
	}
	return release{Version: c.cask, URL: "https://github.com/kadaliao/quickedit/releases/tag/v" + c.cask}, true
}

// caskBehind tells that a release is newer than the app, but Homebrew's
// cask doesn't have it yet.
func (c updateCheck) caskBehind() bool {
	return c.brew && newerVersion(c.latest.Version, c.current) && !newerVersion(c.cask, c.current)
}

// checkForUpdate asks GitHub for the newest release, and for the version
// of the cask when Homebrew installed the app.
func checkForUpdate(ctx context.Context, current string, brew bool) (updateCheck, error) {
	c := updateCheck{current: current, brew: brew}
	b, err := fetchGitHub(ctx, releaseAPI, "application/vnd.github+json")
	if err != nil {
		return c, err
	}
	if c.latest, err = parseRelease(b); err != nil {
		return c, err
	}
	if brew {
		b, err := fetchGitHub(ctx, caskAPI, "application/vnd.github.raw")
		if err != nil {
			return c, err
		}
		if c.cask = parseCaskVersion(string(b)); c.cask == "" {
			return c, errors.New("homebrew cask: no version")
		}
	}
	return c, nil
}

var updateClient = sync.OnceValue(func() *http.Client {
	// Through the proxy of the system's settings, as an app started by
	// the Finder has no proxy in its environment.
	return &http.Client{Transport: netproxy.Transport(), Timeout: 20 * time.Second}
})

func fetchGitHub(ctx context.Context, url, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "QuickEdit")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := updateClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func parseRelease(b []byte) (release, error) {
	var r struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return release{}, fmt.Errorf("GitHub release: %w", err)
	}
	v := strings.TrimPrefix(r.TagName, "v")
	if v == "" || r.HTMLURL == "" {
		return release{}, errors.New("GitHub release: no tag")
	}
	return release{Version: v, URL: r.HTMLURL, Highlights: highlights(r.Body)}, nil
}

var (
	caskVersionLine = regexp.MustCompile(`(?m)^\s*version\s+"([^"]+)"`)
	highlightLine   = regexp.MustCompile(`^[-*]\s+\*\*([^*]+)\*\*`)
)

func parseCaskVersion(rb string) string {
	if m := caskVersionLine.FindStringSubmatch(rb); m != nil {
		return m[1]
	}
	return ""
}

// highlights returns the titles of the points of release notes, which
// start "- **title**", up to the first heading after them (## 安装).
func highlights(notes string) []string {
	var hs []string
	for _, line := range strings.Split(notes, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") && len(hs) > 0 {
			break
		}
		if m := highlightLine.FindStringSubmatch(line); m != nil {
			hs = append(hs, strings.TrimRight(strings.TrimSpace(m[1]), "：:"))
		}
	}
	return hs
}

// newerVersion reports whether version a, as 0.10.1, comes after b. A
// part that isn't a number counts as 0, and a suffix as -beta is ignored.
func newerVersion(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := range max(len(pa), len(pb)) {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func versionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-")
	v, _, _ = strings.Cut(v, "+")
	var parts []int
	for _, s := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(s)
		parts = append(parts, n)
	}
	return parts
}

// fromHomebrew reports whether Homebrew's cask installed this app.
func fromHomebrew() bool {
	exe, err := os.Executable()
	if err != nil || !strings.HasPrefix(exe, "/Applications/QuickEdit.app/") {
		return false
	}
	for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
		if _, err := os.Stat(filepath.Join(prefix, "Caskroom", "quickedit")); err == nil {
			return true
		}
	}
	return false
}

// startUpdateChecks checks for updates while the app runs: a little after
// it starts, then once a day, unless the user turned it off in the App
// menu. Builds being tried out or measured don't check.
func startUpdateChecks() {
	if !installed() {
		return
	}
	go func() {
		time.Sleep(10 * time.Second)
		for {
			autoCheckForUpdate()
			time.Sleep(time.Hour)
		}
	}()
}

// offering tells that the app tells, or waits to tell, of an update.
var offering atomic.Bool

func autoCheckForUpdate() {
	var off bool
	var last time.Time
	mygo.RunOnMain(func() { off, last = settings.updateCheckOff, settings.lastUpdateCheck })
	// Since the last check by the clock, as the Mac may have slept.
	if off || offering.Load() || time.Since(last) < updateInterval {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, err := checkForUpdate(ctx, mygo.App.Version(), fromHomebrew())
	if err != nil {
		log.Printf("check for updates: %v", err)
		return
	}
	var skipped string
	mygo.RunOnMain(func() {
		settings.lastUpdateCheck = time.Now()
		skipped = settings.skippedVersion
		saveSettings()
	})
	if r, ok := c.offer(); ok && r.Version != skipped && offering.CompareAndSwap(false, true) {
		offerWhenActive(c)
	}
}

// offerWhenActive tells of the update once the user is in QuickEdit, as
// the app started at login waits in the background.
func offerWhenActive(c updateCheck) {
	mygo.RunOnMain(func() {
		if mygo.App.IsActive() {
			go offerUpdate(c, false)
			return
		}
		var off func()
		off = mygo.App.OnDidBecomeActive(func() {
			off()
			// After the window the app was activated for shows.
			time.AfterFunc(time.Second, func() { offerUpdate(c, false) })
		})
	})
}

// checking tells that the App menu's Check for Updates is under way.
var checking atomic.Bool

// checkForUpdateNow is the App menu's Check for Updates: it tells of the
// newest release even when the user skipped it, or that there is none.
func checkForUpdateNow() {
	if !checking.CompareAndSwap(false, true) {
		return
	}
	defer checking.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, err := checkForUpdate(ctx, mygo.App.Version(), fromHomebrew())
	if err != nil {
		mygo.Dialog.Message(mygo.MessageOptions{Parent: mygo.FocusedWindow(), Type: mygo.MessageWarning,
			Message: tr("无法检查更新", "Can't Check for Updates"),
			Detail:  tr("连不上 GitHub：", "GitHub didn't answer: ") + err.Error()})
		return
	}
	mygo.RunOnMain(func() {
		settings.lastUpdateCheck = time.Now()
		saveSettings()
	})
	if _, ok := c.offer(); ok {
		if offering.CompareAndSwap(false, true) {
			offerUpdate(c, true)
		}
		return
	}
	detail := fmt.Sprintf(tr("你用的 %s 是最新版本。", "%s is the newest version."), c.current)
	if c.caskBehind() {
		detail = fmt.Sprintf(tr("%s 刚刚发布，Homebrew 几个小时内会更新到它，到时候再检查一次。",
			"%s was just released. Homebrew will have it within a few hours; check again then."), c.latest.Version)
	}
	mygo.Dialog.Message(mygo.MessageOptions{Parent: mygo.FocusedWindow(), Type: mygo.MessageInfo,
		Message: tr("QuickEdit 已是最新版本", "QuickEdit Is Up to Date"), Detail: detail})
}

// offerUpdate shows the dialog of an update, on the focused window. The
// dialog the app shows by itself takes no Enter, which may be the user's
// typing.
func offerUpdate(c updateCheck, asked bool) {
	defer offering.Store(false)
	r, _ := c.offer()
	detail := fmt.Sprintf(tr("你现在用的是 %s。", "You have %s."), c.current)
	if len(r.Highlights) > 0 {
		detail += tr("这一版：", " What's new: ") + strings.Join(r.Highlights, tr("、", ", ")) + tr("。", ".")
	}
	first := tr("下载", "Download")
	if c.brew {
		detail += tr("\n\n它是用 Homebrew 装的，在终端运行下面的命令更新：\n\n", "\n\nHomebrew installed it. Update it in Terminal with:\n\n") + brewUpgrade
		first = tr("拷贝命令", "Copy Command")
	}
	def := -1
	if asked {
		def = 0
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        mygo.FocusedWindow(),
		Type:          mygo.MessageInfo,
		Message:       fmt.Sprintf(tr("QuickEdit %s 已发布", "QuickEdit %s Is Available"), r.Version),
		Detail:        detail,
		Buttons:       []string{first, tr("以后再说", "Later"), tr("跳过这个版本", "Skip This Version")},
		DefaultButton: def,
		CancelButton:  1,
	})
	if err != nil {
		return
	}
	switch res.Button {
	case 0:
		if c.brew {
			mygo.Clipboard.WriteText(brewUpgrade)
		} else if err := mygo.Shell.OpenExternal(r.URL); err != nil {
			log.Printf("open %s: %v", r.URL, err)
		}
	case 2:
		mygo.RunOnMain(func() {
			settings.skippedVersion = r.Version
			saveSettings()
		})
	}
}

func toggleUpdateCheck(item *mygo.MenuItem, _ *mygo.Window) {
	// The item is checked or unchecked already, as it was chosen.
	settings.updateCheckOff = !item.Checked
	saveSettings()
}

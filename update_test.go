package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.5.0", "0.4.0", true},
		{"0.4.0", "0.4.0", false},
		{"0.4.0", "0.5.0", false},
		{"0.10.0", "0.9.9", true},
		{"v1.0", "0.99.99", true},
		{"0.4.1", "0.4", true},
		{"0.4", "0.4.0", false},
		{"0.5.0-beta", "0.4.0", true},
		{"0.5.0-beta", "0.5.0", false},
		{"", "0.4.0", false},
	} {
		if got := newerVersion(c.a, c.b); got != c.want {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// notes are the notes of QuickEdit 0.4.0, shortened.
const notes = "- **文件树**：打开文件夹后，左边的边栏从一层列表改成了文件树。\n" +
	"  当前文件在树里高亮。\n" +
	"- **文件图标**：Go、JS/TS 各有图标。\n" +
	"- **键盘操作**：↑↓ 选择、→← 展开收起。\n\n" +
	"## 安装\n\n```sh\nbrew install --cask kadaliao/tap/quickedit\n```\n\n- **不在要点里**：安装说明\n"

func TestHighlights(t *testing.T) {
	want := []string{"文件树", "文件图标", "键盘操作"}
	if got := highlights(notes); !slices.Equal(got, want) {
		t.Errorf("highlights = %q, want %q", got, want)
	}
	if got := highlights("修好了一个问题。"); got != nil {
		t.Errorf("highlights of plain notes = %q", got)
	}
}

func TestParseCaskVersion(t *testing.T) {
	rb := "cask \"quickedit\" do\n  version \"0.4.0\"\n  sha256 \"12ee\"\n\n  url \"https://github.com/kadaliao/quickedit/releases/download/v#{version}/x.dmg\"\n"
	if got := parseCaskVersion(rb); got != "0.4.0" {
		t.Errorf("parseCaskVersion = %q, want 0.4.0", got)
	}
	if got := parseCaskVersion("cask \"x\" do\nend\n"); got != "" {
		t.Errorf("parseCaskVersion without version = %q", got)
	}
}

// fakeGitHub serves a release and a cask in place of GitHub's API.
func fakeGitHub(t *testing.T, tag, cask string, status int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("%s: no User-Agent, which GitHub's API requires", r.URL.Path)
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		switch r.URL.Path {
		case "/release":
			json.NewEncoder(w).Encode(map[string]string{
				"tag_name": "v" + tag,
				"html_url": "https://github.com/kadaliao/quickedit/releases/tag/v" + tag,
				"body":     notes,
			})
		case "/cask":
			if r.Header.Get("Accept") != "application/vnd.github.raw" {
				t.Errorf("cask asked as %q, not raw", r.Header.Get("Accept"))
			}
			w.Write([]byte("cask \"quickedit\" do\n  version \"" + cask + "\"\nend\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	oldRelease, oldCask := releaseAPI, caskAPI
	releaseAPI, caskAPI = srv.URL+"/release", srv.URL+"/cask"
	t.Cleanup(func() { releaseAPI, caskAPI = oldRelease, oldCask })
}

func TestCheckForUpdate(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name       string
		tag, cask  string
		current    string
		brew       bool
		offer      string // "" for none
		highlights bool
		caskBehind bool
	}{
		{name: "newer release", tag: "0.5.0", current: "0.4.0", offer: "0.5.0", highlights: true},
		{name: "newest", tag: "0.4.0", current: "0.4.0"},
		{name: "build ahead of the release", tag: "0.4.0", current: "0.5.0"},
		{name: "brew, cask has it", tag: "0.5.0", cask: "0.5.0", current: "0.4.0", brew: true, offer: "0.5.0", highlights: true},
		{name: "brew, cask behind", tag: "0.5.0", cask: "0.4.0", current: "0.4.0", brew: true, caskBehind: true},
		{name: "brew, cask between", tag: "0.6.0", cask: "0.5.0", current: "0.4.0", brew: true, offer: "0.5.0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fakeGitHub(t, c.tag, c.cask, http.StatusOK)
			check, err := checkForUpdate(ctx, c.current, c.brew)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := check.offer()
			if got := map[bool]string{true: r.Version}[ok]; got != c.offer {
				t.Fatalf("offer = %q, want %q", got, c.offer)
			}
			if ok && r.URL != "https://github.com/kadaliao/quickedit/releases/tag/v"+c.offer {
				t.Errorf("URL = %q", r.URL)
			}
			if got := len(r.Highlights) > 0; got != c.highlights {
				t.Errorf("highlights = %q", r.Highlights)
			}
			if got := check.caskBehind(); got != c.caskBehind {
				t.Errorf("caskBehind = %v, want %v", got, c.caskBehind)
			}
		})
	}
}

func TestCheckForUpdateFails(t *testing.T) {
	fakeGitHub(t, "0.5.0", "0.5.0", http.StatusForbidden)
	if _, err := checkForUpdate(context.Background(), "0.4.0", false); err == nil {
		t.Fatal("no error when GitHub refuses")
	}
}

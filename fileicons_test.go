package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Every icon parses, and every name the maps give has an icon.
func TestFileIcons(t *testing.T) {
	for n := range fileIcons {
		if _, err := ui.ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">` + fileIcons[n] + `</svg>`)); err != nil {
			t.Errorf("icon %s: %v", n, err)
		}
	}
	for _, m := range []map[string]string{iconByName, iconByExt} {
		for k, n := range m {
			if _, ok := fileIcons[n]; !ok {
				t.Errorf("%s: no icon %q", k, n)
			}
		}
	}
	tests := map[string]string{
		"main.go": "go", "README.md": "readme", "readme": "readme", "LICENSE": "license", "Makefile": "make",
		"notes.MD": "markdown", "a.d.ts": "ts", "tsconfig.base.json": "ts", ".env.local": "env", "x.unknown": "file",
		"Dockerfile.dev": "docker", ".prettierrc": "config", "go.sum": "go", "photo.JPG": "image", "noext": "file",
	}
	for name, want := range tests {
		if got := iconName(name, false, false); got != want {
			t.Errorf("iconName(%q) = %q, want %q", name, got, want)
		}
	}
	if iconName(".github", true, true) != "folder-dot-open" || iconName("src", true, false) != "folder" {
		t.Error("folder icons")
	}
}

// QE_ICON_SHEET=/path/sheet.png go test -run TestIconSheet draws every
// icon, at the sidebar's size and larger, on light and dark, to look at
// them.
func TestIconSheet(t *testing.T) {
	out := os.Getenv("QE_ICON_SHEET")
	if out == "" {
		t.Skip("QE_ICON_SHEET not set")
	}
	var names []string
	for n := range fileIcons {
		names = append(names, n)
	}
	slices.Sort(names)
	sheet := func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Fill().Background(t.Background).Padding(12).Gap(10).Wrap().AlignContent(ui.Start).Children(func() {
			for _, n := range names {
				ui.Column(c.Key(n)).Width(84).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Image(c, iconNamed(n)).Size(16, 16)
						ui.Image(c, iconNamed(n)).Size(32, 32)
					})
					ui.Text(c, n).FontSize(10).TextColor(t.TextMuted)
				})
			}
		})
	}
	tt := ui.NewTester(sheet, 960, 520)
	tt.SetScale(2)
	tt.Frame()
	light := tt.Image()
	tt.SetDark(true)
	tt.Frame()
	dark := tt.Image()
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, stack(light, dark))
}

// stack puts b below a.
func stack(a, b *image.RGBA) *image.RGBA {
	r := a.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()+b.Bounds().Dy()))
	draw.Draw(out, r, a, image.Point{}, draw.Src)
	draw.Draw(out, r.Add(image.Pt(0, r.Dy())), b, image.Point{}, draw.Src)
	return out
}

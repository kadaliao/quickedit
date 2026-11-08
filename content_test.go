package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

func mustEncode(t *testing.T, s string, enc interface{ Bytes([]byte) ([]byte, error) }) []byte {
	t.Helper()
	b, err := enc.Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Each file reads as its text and writes back byte for byte.
func TestRoundTrip(t *testing.T) {
	const zh = "第一行：中文\n第二行 English\n"
	tests := []struct {
		name string
		file []byte
		enc  textEncoding
	}{
		{"utf8", []byte(zh), encUTF8},
		{"utf8 crlf", []byte(strings.ReplaceAll(zh, "\n", "\r\n")), encUTF8},
		{"utf8 bom", append([]byte{0xEF, 0xBB, 0xBF}, zh...), encUTF8BOM},
		{"gbk", mustEncode(t, zh, simplifiedchinese.GBK.NewEncoder()), encGB18030},
		{"utf16le", mustEncode(t, zh, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()), encUTF16LE},
		{"utf16be", mustEncode(t, zh, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()), encUTF16BE},
		{"empty", nil, encUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := readContent(bytes.NewReader(tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if c.binary {
				t.Fatal("read as binary")
			}
			if c.enc != tt.enc {
				t.Fatalf("encoding %v, want %v", c.enc, tt.enc)
			}
			if len(tt.file) > 0 && c.snap.String() != zh {
				t.Fatalf("text %q, want %q", c.snap.String(), zh)
			}
			var out bytes.Buffer
			if err := writeContent(&out, c); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), tt.file) {
				t.Fatalf("wrote % x\nwant  % x", out.Bytes(), tt.file)
			}
		})
	}
}

// An edit in GBK saves in GBK.
func TestEditKeepsEncoding(t *testing.T) {
	file := mustEncode(t, "你好\n", simplifiedchinese.GBK.NewEncoder())
	c, _ := readContent(bytes.NewReader(file))
	c.buf.Edit(editor.Edit{Start: c.buf.Len(), End: c.buf.Len(), Text: "世界\n"})
	c.snap = c.buf.Snapshot()
	var out bytes.Buffer
	if err := writeContent(&out, c); err != nil {
		t.Fatal(err)
	}
	if want := mustEncode(t, "你好\n世界\n", simplifiedchinese.GBK.NewEncoder()); !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("wrote % x, want % x", out.Bytes(), want)
	}
}

func TestBinary(t *testing.T) {
	file := append([]byte("\x7fELF\x00\x01"), bytes.Repeat([]byte{0xAB}, 40)...)
	c, err := readContent(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	if !c.binary || c.truncated {
		t.Fatalf("binary %v truncated %v", c.binary, c.truncated)
	}
	if !strings.HasPrefix(c.snap.String(), "00000000  7f 45 4c 46 00 01 ab ab  ab ab") {
		t.Fatalf("hex dump %q", c.snap.Line(0))
	}
	if err := writeContent(&bytes.Buffer{}, c); err != errBinary {
		t.Fatalf("writing a binary file: %v", err)
	}

	// Only the start of a large binary file is read.
	big := append([]byte{0}, bytes.Repeat([]byte{1}, 3*hexLimit)...)
	c, _ = readContent(bytes.NewReader(big))
	if !c.binary || !c.truncated || c.snap.LineCount() > hexLimit/16+1 {
		t.Fatalf("binary %v truncated %v lines %d", c.binary, c.truncated, c.snap.LineCount())
	}
}

func TestEqualContent(t *testing.T) {
	a, _ := readContent(strings.NewReader("abc\n"))
	b := a
	b.buf.Edit(editor.Edit{Start: 0, End: 1, Text: "x"})
	b.snap = b.buf.Snapshot()
	if equalContent(a, b) {
		t.Fatal("abc and xbc are equal")
	}
	b.buf.Edit(editor.Edit{Start: 0, End: 1, Text: "a"})
	b.snap = b.buf.Snapshot()
	if !equalContent(a, b) {
		t.Fatal("undoing back to the file's text leaves it edited")
	}
}

func TestShebang(t *testing.T) {
	for line, want := range map[string]string{
		"#!/usr/bin/env python3\n":   "Python",
		"#!/bin/bash\n":              "Shell",
		"#!/usr/bin/env -S node --x": "JavaScript",
		"no shebang\n":               "",
	} {
		got := ""
		if l := languageOfShebang(editor.NewBuffer(line).Snapshot()); l != nil {
			got = l.Name
		}
		if got != want {
			t.Errorf("%q: %q, want %q", line, got, want)
		}
	}
}

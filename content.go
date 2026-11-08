package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
	"unsafe"

	"github.com/egoist/mygo/ui/editor"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// content is what a document holds: its text, as a snapshot of the buffer
// its window edits, and how the file stores it.
type content struct {
	// buf is the buffer of the text. read makes it, off the main thread,
	// and the window then edits it.
	buf  *editor.Buffer
	snap editor.Snapshot
	enc  textEncoding
	// binary files show as a hex dump, read-only, and are never written.
	binary    bool
	truncated bool // the hex dump shows only the start of the file
}

type textEncoding int

const (
	encUTF8 textEncoding = iota
	encUTF8BOM
	encUTF16LE
	encUTF16BE
	encGB18030
)

func (e textEncoding) String() string {
	return [...]string{"UTF-8", "UTF-8 BOM", "UTF-16 LE", "UTF-16 BE", "GB18030"}[e]
}

func (e textEncoding) codec() encoding.Encoding {
	switch e {
	case encUTF16LE:
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	case encUTF16BE:
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	case encGB18030:
		return simplifiedchinese.GB18030
	}
	return nil
}

var bomUTF8 = []byte{0xEF, 0xBB, 0xBF}

// hexLimit is how much of a binary file the hex dump shows.
const hexLimit = 1 << 20

// sniffSize is how much of a file decides whether it is text, as Git does.
const sniffSize = 8000

func readContent(r io.Reader) (content, error) {
	// Into one buffer the size of the file: a 44 MB log then is copied
	// once, by the reads, and its text shares the buffer's bytes.
	data := bytes.NewBuffer(make([]byte, 0, sizeHint(r)))
	if _, err := io.CopyN(data, r, sniffSize); err != nil && err != io.EOF {
		return content{}, err
	}
	if looksBinary(data.Bytes()) {
		// Only the start of a binary file is read: a video opened by
		// mistake must not fill the memory.
		if _, err := data.ReadFrom(io.LimitReader(r, hexLimit-int64(data.Len())+1)); err != nil {
			return content{}, err
		}
		b := data.Bytes()
		truncated := len(b) > hexLimit
		if truncated {
			b = b[:hexLimit]
		}
		buf := editor.NewBuffer(hexDump(b))
		return content{buf: buf, snap: buf.Snapshot(), binary: true, truncated: truncated}, nil
	}
	if _, err := data.ReadFrom(r); err != nil {
		return content{}, err
	}
	b := data.Bytes()
	text, enc, ok := decode(b)
	if !ok {
		buf := editor.NewBuffer(hexDump(b[:min(len(b), hexLimit)]))
		return content{buf: buf, snap: buf.Snapshot(), binary: true, truncated: len(b) > hexLimit}, nil
	}
	buf := editor.NewBuffer(text)
	return content{buf: buf, snap: buf.Snapshot(), enc: enc}, nil
}

// sizeHint returns how many bytes reading r takes, as read with
// bytes.Buffer.ReadFrom: the size of a file, and some room for the read
// that finds its end.
func sizeHint(r io.Reader) int {
	if f, ok := r.(interface{ Stat() (fs.FileInfo, error) }); ok {
		if info, err := f.Stat(); err == nil && info.Mode().IsRegular() && info.Size() < 1<<31 {
			return int(info.Size()) + bytes.MinRead
		}
	}
	return 64 << 10
}

// looksBinary reports whether the start of a file holds a NUL byte, as
// Git decides, unless it is UTF-16, which has them in every ASCII letter.
func looksBinary(head []byte) bool {
	if bytes.HasPrefix(head, []byte{0xFF, 0xFE}) || bytes.HasPrefix(head, []byte{0xFE, 0xFF}) {
		return false
	}
	return bytes.IndexByte(head, 0) >= 0
}

// decode returns the text of a file and its encoding: UTF-8, with or
// without a BOM, UTF-16 with a BOM, or else GB18030, which reads GBK and
// GB2312 too. ok is false when none of them reads it. UTF-8 text shares
// b's bytes, which must not change afterwards.
func decode(b []byte) (text string, enc textEncoding, ok bool) {
	switch {
	case bytes.HasPrefix(b, bomUTF8):
		if validUTF8(b[3:]) {
			return unsafe.String(unsafe.SliceData(b[3:]), len(b)-3), encUTF8BOM, true
		}
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return decodeWith(b, encUTF16LE)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return decodeWith(b, encUTF16BE)
	}
	if validUTF8(b) {
		return unsafe.String(unsafe.SliceData(b), len(b)), encUTF8, true
	}
	return decodeWith(b, encGB18030)
}

// validUTF8 is utf8.Valid, on every core for a long text: cut before
// bytes that start a rune, each part is valid UTF-8 only if the whole is.
func validUTF8(b []byte) bool {
	const part = 1 << 20
	workers := min(runtime.GOMAXPROCS(0), len(b)/part)
	if workers < 2 {
		return utf8.Valid(b)
	}
	var bad atomic.Bool
	var wg sync.WaitGroup
	start := 0
	for i := 1; i <= workers; i++ {
		end := len(b)
		if i < workers {
			end = len(b) * i / workers
			for end > start && !utf8.RuneStart(b[end]) {
				end--
			}
		}
		part := b[start:end]
		wg.Go(func() {
			if !utf8.Valid(part) {
				bad.Store(true)
			}
		})
		start = end
	}
	wg.Wait()
	return !bad.Load()
}

func decodeWith(b []byte, enc textEncoding) (string, textEncoding, bool) {
	out, err := enc.codec().NewDecoder().Bytes(b)
	if err != nil || bytes.ContainsRune(out, utf8.RuneError) {
		return "", enc, false
	}
	return string(out), enc, true
}

var errBinary = errors.New("二进制文件不能保存 (binary files can't be saved)")

func writeContent(w io.Writer, c content) error {
	if c.binary {
		return errBinary
	}
	switch c.enc {
	case encUTF8:
		_, err := c.snap.WriteTo(w)
		return err
	case encUTF8BOM:
		if _, err := w.Write(bomUTF8); err != nil {
			return err
		}
		_, err := c.snap.WriteTo(w)
		return err
	}
	tw := transform.NewWriter(w, c.enc.codec().NewEncoder())
	if _, err := c.snap.WriteTo(tw); err != nil {
		return err
	}
	return tw.Close()
}

// equalContent tells whether two contents would write the same file, so
// that undoing back to what the file has leaves the document unedited.
func equalContent(a, b content) bool {
	if a.enc != b.enc || a.binary != b.binary || a.snap.Len() != b.snap.Len() {
		return false
	}
	if a.buf == b.buf && a.snap.Version() == b.snap.Version() {
		return true
	}
	eq, off := true, 0
	a.snap.Chunks(0, a.snap.Len(), func(s string) bool {
		if b.snap.Slice(off, off+len(s)) != s {
			eq = false
			return false
		}
		off += len(s)
		return true
	})
	return eq
}

// hexDump formats b as hexdump -C does.
func hexDump(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b)/16*79 + 80)
	for off := 0; off < len(b); off += 16 {
		line := b[off:min(off+16, len(b))]
		fmt.Fprintf(&sb, "%08x  ", off)
		for i := range 16 {
			if i < len(line) {
				fmt.Fprintf(&sb, "%02x ", line[i])
			} else {
				sb.WriteString("   ")
			}
			if i == 7 {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(" |")
		for _, c := range line {
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}

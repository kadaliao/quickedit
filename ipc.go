package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// The qe command hands QuickEdit its files through a socket, as the
// system's way, LaunchServices, takes 40 to 60 ms to pass a running app a
// file. qe writes each absolute path followed by a NUL byte, then one more
// NUL, and QuickEdit answers "ok\n" once it has asked for them to open.
// When nothing listens, qe opens them with open -a.
//
// macOS lets only the app in front, or LaunchServices, bring another app
// to the front, so qe then asks LaunchServices to activate QuickEdit with
// open -b, while the files load: the windows show in front once both are
// done, about 50 ms sooner than when open hands over the files.

// socketPath is where QuickEdit listens; scripts/qe knows it too.
func socketPath() string { return filepath.Join(userData(), "qe.sock") }

// listen serves qe while the app runs. A second copy of QuickEdit leaves
// the socket to the first.
func listen() {
	path := socketPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if c, err := net.Dial("unix", path); err == nil {
		c.Close()
		return
	}
	// Left by a QuickEdit that crashed.
	os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		log.Printf("qe socket: %v", err)
		return
	}
	os.Chmod(path, 0o600)
	mygo.App.OnQuit(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go serve(c)
		}
	}()
}

// pending counts the files of qe still opening, while the activation qe
// asks for comes as a reopen, which must not make an untitled document.
var pending atomic.Int32

func serve(c net.Conn) {
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(c)
	for {
		p, err := r.ReadString(0)
		if err != nil {
			return
		}
		p = p[:len(p)-1]
		if p == "" {
			break
		}
		if filepath.IsAbs(p) {
			requested.Store(true)
			pending.Add(1)
			go func() {
				openAnyFile(filepath.Clean(p))
				pending.Add(-1)
			}()
		}
	}
	c.Write([]byte("ok\n"))
}

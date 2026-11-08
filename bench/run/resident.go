package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// runResident measures opening files through LaunchServices (open -a), as a
// double click in Finder does: a cold launch, then files handed to the app
// while it runs.
func runResident(root string, apps, files []string, runs int) {
	for _, a := range apps {
		bundle := filepath.Join(root, "build", "QE-"+a+".app")
		exe := "qe-" + a
		logPath := filepath.Join(os.TempDir(), "qe-"+a+".log")
		var cold []float64
		warm := map[string][]float64{}
		for r := range runs + 1 { // the first launch warms up, not measured
			exec.Command("pkill", "-x", exe).Run()
			time.Sleep(500 * time.Millisecond)
			os.Remove(logPath)
			began := time.Now()
			if err := exec.Command("open", "-a", bundle, "--env", "QE_LOG="+logPath, filepath.Join(root, "files", "small.go")).Run(); err != nil {
				log.Fatal(err)
			}
			pid := 0
			for pid == 0 && time.Since(began) < 30*time.Second {
				if out, err := exec.Command("pgrep", "-x", exe).Output(); err == nil {
					pid, _ = strconv.Atoi(strings.Fields(string(out))[0])
				}
			}
			for windowCount(pid) < 1 && time.Since(began) < 30*time.Second {
				time.Sleep(2 * time.Millisecond)
			}
			if r > 0 {
				cold = append(cold, ms(time.Since(began)))
			}
			time.Sleep(time.Second)
			for _, f := range files {
				n := windowCount(pid)
				began := time.Now()
				exec.Command("open", "-a", bundle, filepath.Join(root, "files", f)).Run()
				for windowCount(pid) <= n && time.Since(began) < 30*time.Second {
					time.Sleep(2 * time.Millisecond)
				}
				if r > 0 {
					warm[f] = append(warm[f], ms(time.Since(began)))
				}
				time.Sleep(500 * time.Millisecond)
			}
			b, _ := os.ReadFile(logPath)
			if r > 0 {
				log.Printf("run %d %s: in-app open→first frame: %s", r, a, strings.ReplaceAll(strings.TrimSpace(string(b)), "\n", ", "))
			}
		}
		exec.Command("pkill", "-x", exe).Run()
		fmt.Printf("%-7s cold launch via open -a: %s ms\n", a, seriesOf(cold))
		for _, f := range files {
			fmt.Printf("%-7s already running, open %-9s %s ms\n", a, f, seriesOf(warm[f]))
		}
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func seriesOf(v []float64) series {
	ss := make([]sample, len(v))
	for i, x := range v {
		ss[i].StartupMs = x
	}
	return stat(ss, func(s sample) float64 { return s.StartupMs })
}

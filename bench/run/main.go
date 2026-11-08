// Command run starts each editor app of bench/bin on each test file in turn,
// and measures how long until the file shows, highlighted, and the memory it
// then takes. See ../README.md.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type sample struct {
	StartupMs, HighlightMs, FootprintMB float64
	DrawnMs, OnScreenMs                float64 // the two parts of StartupMs, which is the later of them
}

func main() {
	runs := flag.Int("runs", 10, "runs of each app on each file")
	appsFlag := flag.String("apps", "", "comma-separated apps (default: every bin/bench-*)")
	filesFlag := flag.String("files", "small.go,big.go,doc.md", "comma-separated test files")
	out := flag.String("out", "", "write the samples as JSON here")
	resident := flag.Bool("resident", false, "measure opening files through LaunchServices instead (build/QE-*.app)")
	flag.Parse()
	log.SetFlags(0)

	root, _ := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), ".."))
	if _, err := os.Stat(filepath.Join(root, "files")); err != nil {
		root, _ = os.Getwd()
	}
	var apps []string
	if *appsFlag != "" {
		apps = strings.Split(*appsFlag, ",")
	} else {
		bins, _ := filepath.Glob(filepath.Join(root, "bin", "bench-*"))
		for _, b := range bins {
			apps = append(apps, strings.TrimPrefix(filepath.Base(b), "bench-"))
		}
	}
	files := strings.Split(*filesFlag, ",")
	if *resident {
		runResident(root, apps, files, *runs)
		return
	}
	bin := func(a string) string { return filepath.Join(root, "bin", "bench-"+a) }
	file := func(f string) string { return filepath.Join(root, "files", f) }

	load, _ := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	log.Printf("load average %s", strings.TrimSpace(string(load)))

	// The first start of a new binary is checked by the system and its pages
	// are not cached yet: one start of each, not measured.
	for _, a := range apps {
		if _, err := measure(bin(a), file("small.go")); err != nil {
			log.Fatalf("warm-up %s: %v", a, err)
		}
	}

	samples := map[string]map[string][]sample{}
	for r := range *runs {
		for _, f := range files {
			for i := range apps {
				a := apps[(i+r)%len(apps)] // take turns, starting with a different app each run
				s, err := measure(bin(a), file(f))
				if err != nil {
					log.Printf("run %d %s %s: %v", r+1, a, f, err)
					continue
				}
				if samples[a] == nil {
					samples[a] = map[string][]sample{}
				}
				samples[a][f] = append(samples[a][f], s)
				log.Printf("run %d %-7s %-9s startup %6.0f ms (drawn %4.0f, on screen %4.0f)  highlighted %6.0f ms  %5.0f MB", r+1, a, f, s.StartupMs, s.DrawnMs, s.OnScreenMs, s.HighlightMs, s.FootprintMB)
			}
		}
	}

	fmt.Printf("\nmedian (min-max) of %d runs\n", *runs)
	fmt.Printf("%-8s %-9s %-20s %-20s %-10s %s\n", "app", "file", "startup ms", "highlighted ms", "memory MB", "binary")
	for _, a := range apps {
		info, _ := os.Stat(bin(a))
		for _, f := range files {
			ss := samples[a][f]
			if len(ss) == 0 {
				fmt.Printf("%-8s %-9s failed\n", a, f)
				continue
			}
			st := stat(ss, func(s sample) float64 { return s.StartupMs })
			hl := stat(ss, func(s sample) float64 { return s.HighlightMs })
			mem := stat(ss, func(s sample) float64 { return s.FootprintMB })
			fmt.Printf("%-8s %-9s %-20s %-20s %-10.0f %.1f MB\n", a, f, st, hl, mem.med, float64(info.Size())/1e6)
		}
	}
	if *out != "" {
		b, _ := json.MarshalIndent(map[string]any{"date": time.Now().UTC().Format(time.RFC3339), "runs": *runs, "samples": samples}, "", "  ")
		os.WriteFile(*out, append(b, '\n'), 0o644)
	}
}

type series struct{ med, min, max float64 }

func (s series) String() string { return fmt.Sprintf("%.0f (%.0f-%.0f)", s.med, s.min, s.max) }

func stat(ss []sample, f func(sample) float64) series {
	v := make([]float64, len(ss))
	for i, s := range ss {
		v[i] = f(s)
	}
	slices.Sort(v)
	return series{v[len(v)/2], v[0], v[len(v)-1]}
}

// measure starts an app on a file, and times its first frame with the file,
// its first highlighted frame and its window coming on screen.
func measure(bin, path string) (sample, error) {
	began := time.Now()
	cmd := exec.Command(bin, path, "-_NS_4445425547", "NO")
	cmd.Stderr = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return sample{}, err
	}
	if err := cmd.Start(); err != nil {
		return sample{}, err
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
		time.Sleep(300 * time.Millisecond)
	}()
	shown := make(chan time.Duration, 1)
	go func() {
		for time.Since(began) < time.Minute {
			if windowOnScreen(cmd.Process.Pid) {
				shown <- time.Since(began)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		shown <- 0
	}()
	lines := make(chan string, 8)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	wait := func(want string) (time.Duration, error) {
		guard := time.After(time.Minute)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					return 0, fmt.Errorf("exited before %q", want)
				}
				if strings.TrimSpace(l) == want {
					return time.Since(began), nil
				}
			case <-guard:
				return 0, fmt.Errorf("no %q within a minute", want)
			}
		}
	}
	drawn, err := wait("ready")
	if err != nil {
		return sample{}, err
	}
	hl, err := wait("highlighted")
	if err != nil {
		return sample{}, err
	}
	on := <-shown
	if on == 0 {
		return sample{}, fmt.Errorf("no window on screen")
	}
	time.Sleep(2 * time.Second)
	u, err := usageOf(cmd.Process.Pid)
	if err != nil {
		return sample{}, err
	}
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	return sample{StartupMs: ms(max(drawn, on)), HighlightMs: ms(max(hl, on)), FootprintMB: float64(u.footprint) / 1e6, DrawnMs: ms(drawn), OnScreenMs: ms(on)}, nil
}

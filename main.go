package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const RealP4Win = "C:\\Program Files\\Perforce\\p4.exe"
const RealP4Nix = "/usr/local/bin/p4"
const TEMPLATE = `
----------
Time: %s
Executing: p4 %s

CWD: %s

P4 Environment:

%s

Full Output:

%s

Exit Status: %d

Exec Time: %d ms
`

type Prefs struct {
	P4Path   string `json:"p4Path"`
	LogDir   string `json:"logDir"`
	MaxLines int    `json:"maxLines"`
	Verbose  bool   `json:"verbose"`
}

func defaults() *Prefs {
	var P4Path string
	var LogDir string

	if runtime.GOOS == "windows" {
		P4Path = RealP4Win
		LogDir = "C:\\tmp"
	} else {
		P4Path = RealP4Nix
		LogDir = "/tmp"
	}

	return &Prefs{P4Path: P4Path, LogDir: LogDir, MaxLines: -1, Verbose: false}
}

func readPrefs() *Prefs {
	prefsFilePath := filepath.Join(cwd(), "p4-wrapper.json")
	def := defaults()

	if _, err := os.Stat(prefsFilePath); os.IsNotExist(err) {
		return def
	}

	if data, err := ioutil.ReadFile(prefsFilePath); err == nil {
		var prefs *Prefs
		json.Unmarshal(data, &prefs)

		if strings.TrimSpace(prefs.P4Path) == "" {
			prefs.P4Path = def.P4Path
		}

		if strings.TrimSpace(prefs.LogDir) == "" {
			prefs.LogDir = def.LogDir
		}

		if prefs.MaxLines == 0 {
			prefs.MaxLines = def.MaxLines
		}

		return prefs
	} else {
		log.Fatal(err)
	}

	return def
}

func sel(strs []string, test func(string) bool) (ret []string) {
	for _, s := range strs {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func p4Envs() []string {
	re := regexp.MustCompile("^P4[^=]+=")

	return sel(os.Environ(), func(s string) bool {
		return re.MatchString(s)
	})
}

func cwd() string {
	dir, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func getExitStatus(err error) int {
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return 1
}

func writeToLog(filename, text string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	if _, err = f.WriteString(text); err != nil {
		log.Fatal(err)
	}
}

func lineEndings(s string) string {
	if runtime.GOOS == "windows" {
		re := regexp.MustCompile("\n")
		s = re.ReplaceAllString(s, "\r\n")
	}
	return s
}

func main() {
	prefs := readPrefs()

	if err := os.MkdirAll(prefs.LogDir, 0755); err != nil {
		log.Fatal(err)
	}

	logPath := filepath.Join(prefs.LogDir, "p4-debug.log")

	args := os.Args[1:]

	if prefs.Verbose {
		args = append([]string{"-v", "4"}, args...)
	}

	start := time.Now()
	cmd := exec.Command(prefs.P4Path, args...)
	cmd.Env = os.Environ()

	if stdin, err := cmd.StdinPipe(); err != nil {
		log.Fatal(err)
	} else {
		go func() {
			defer stdin.Close()

			if data, ioerr := ioutil.ReadAll(os.Stdin); ioerr == nil {
				io.WriteString(stdin, string(data))
			} else {
				log.Fatal(ioerr)
			}
		}()
	}

	code := 0
	out, err := cmd.CombinedOutput()

	duration := int64(time.Since(start) / time.Millisecond)

	if err != nil {
		code = getExitStatus(err)
	}

	extended := fmt.Sprintf(lineEndings(TEMPLATE), start.Format(time.RFC3339), strings.Join(args, " "), cwd(), lineEndings(strings.Join(p4Envs(), "\n")), out, code, duration)
	writeToLog(logPath, extended)

	if prefs.MaxLines > 0 {
		lines := strings.Split(string(out), "\n")
		limit := prefs.MaxLines

		if limit > len(lines) {
			limit = len(lines)
		}

		truncated := strings.Join(lines[:limit], "\n")
		fmt.Printf("%s\n", truncated)
	} else {
		fmt.Printf("%s\n", out)
	}

	os.Exit(code)
}

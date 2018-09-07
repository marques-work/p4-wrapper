package main

import (
	"encoding/json"
	"fmt"
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

Exec Time: %d ms
`

type Prefs struct {
	P4Path   string `json:"p4Path"`
	LogDir   string `json:"logDir"`
	MaxLines int    `json:"maxLines"`
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

	return &Prefs{P4Path: P4Path, LogDir: LogDir, MaxLines: -1}
}

func readPrefs() *Prefs {
	prefsFilePath := filepath.Join(thisDir(), "p4-wrapper.json")
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

func thisDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

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
	verboseArgs := append([]string{"-v", "4"}, args...)

	start := time.Now()
	cmd := exec.Command(prefs.P4Path, verboseArgs...)
	duration := int64(time.Since(start) / time.Millisecond)

	code := 0
	out, err := cmd.CombinedOutput()

	if err != nil {
		code = getExitStatus(err)
	}

	cwd, _ := os.Getwd()

	extended := fmt.Sprintf(lineEndings(TEMPLATE), start.Format(time.RFC3339), strings.Join(args, " "), cwd, strings.Join(p4Envs(), "\n"), out, duration)
	writeToLog(logPath, extended)

	if prefs.MaxLines > 0 {
		lines := strings.Split(string(out), "\n")
		limit := prefs.MaxLines

		if limit > len(lines) {
			limit = len(lines)
		} else {
			if len(lines) > 0 && limit != len(lines) {
				lines[0] = fmt.Sprintf("[TRUNCATED OUTPUT - %d lines]\n\n%s", limit, lines[0])
			}
		}

		truncated := strings.Join(lines[:limit], "\n")
		fmt.Printf("%s\n", truncated)
	} else {
		fmt.Printf("%s\n", out)
	}

	os.Exit(code)
}

package main

import (
	"fmt"
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

P4 Environment:

%s

Output:

%s

Exec Time: %d ms
`

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

func realP4() string {
	if runtime.GOOS == "windows" {
		return RealP4Win
	} else {
		return RealP4Nix
	}
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

func main() {
	logPath := filepath.Join(thisDir(), "p4-debug.log")

	args := os.Args[1:]
	verboseArgs := append([]string{"-v", "4"}, args...)

	start := time.Now()
	cmd := exec.Command(realP4(), verboseArgs...)
	duration := int64(time.Since(start) / time.Millisecond)

	code := 0
	out, err := cmd.CombinedOutput()

	if err != nil {
		code = getExitStatus(err)
	}

	extended := fmt.Sprintf(TEMPLATE, start.Format(time.RFC3339), strings.Join(args, " "), strings.Join(p4Envs(), "\n"), out, duration)
	writeToLog(logPath, extended)

	fmt.Printf("%s\n", out)
	os.Exit(code)
}

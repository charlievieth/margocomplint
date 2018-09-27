package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/charlievieth/buildutil"
)

func TestPkg(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func MainPkg(path string) bool {
	name, err := buildutil.ReadPackageName(path, nil)
	return err == nil && name == "main"
}

func BenchmarkInit() bool {
	switch os.Getenv("MARGOCOMPLINT_BENCHMARK") {
	case "1", "t", "T", "true", "TRUE", "True":
		return true
	}
	return false
}

func main() {
	path := os.Getenv("GOSUBL_LINT_FILENAME")
	var args []string
	switch {
	case TestPkg(path):
		args = []string{"test", "-c", "-o", os.DevNull}
	case MainPkg(path):
		args = []string{"build", "-o", os.DevNull}
	default:
		args = []string{"install"}
	}
	if FlagI {
		args = append(args, "-i")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if BenchmarkInit() {
		return
	}
	if err := cmd.Run(); err != nil {
		stderr.WriteTo(os.Stderr)
		os.Exit(1)
	}
	if stdout.Len() != 0 {
		stdout.WriteTo(os.Stdout)
	}
}

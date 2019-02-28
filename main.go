package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charlievieth/buildutil"
)

func TestPkg(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func MainPkg(path string) bool {
	if path == "" {
		return false
	}
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
		args = []string{"test", "-c", "-i", "-o", os.DevNull}
	case MainPkg(path):
		args = []string{"build", "-i"}
	default:
		args = []string{"install", "-i"}
	}
	if BenchmarkInit() {
		return
	}
	out, _ := exec.Command("go", args...).CombinedOutput()
	os.Stdout.Write(out)
}

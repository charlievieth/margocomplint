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
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if BenchmarkInit() {
		return
	}
	cmd.Run()
}

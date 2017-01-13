package main

import (
	"os"
	"os/exec"
	"strings"
)

func isTest() bool {
	return strings.HasSuffix(os.Getenv("GOSUBL_LINT_FILENAME"), "_test.go")
}

func main() {
	var cmd *exec.Cmd
	if isTest() {
		cmd = exec.Command("go", "test", "-c", "-o", os.DevNull)
	} else {
		cmd = exec.Command("go", "build", "-o", os.DevNull)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

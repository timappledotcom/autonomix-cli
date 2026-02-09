package homebrew

import (
	"os/exec"
	"runtime"
	"strings"
)

func IsInstalled() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	
	_, err := exec.LookPath("brew")
	return err == nil
}

func GetBrewPrefix() (string, error) {
	cmd := exec.Command("brew", "--prefix")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

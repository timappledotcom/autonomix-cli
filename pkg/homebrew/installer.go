package homebrew

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func SearchFormula(appName string) (string, error) {
	cmd := exec.Command("brew", "search", appName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == appName {
			return appName, nil
		}
	}
	
	return "", fmt.Errorf("no formula found")
}

func InstallOfficial(formulaName string) error {
	cmd := exec.Command("brew", "install", formulaName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install failed: %w", err)
	}
	
	return nil
}

func UpdateWithBrew(appName string) error {
	cmd := exec.Command("brew", "upgrade", appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	
	return nil
}

func IsInstalledViaBrew(appName string) bool {
	cmd := exec.Command("brew", "list", appName)
	err := cmd.Run()
	return err == nil
}

func GetInstalledVersion(appName string) (string, error) {
	cmd := exec.Command("brew", "list", "--versions", appName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// Output format: "appname version"
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	
	return "", fmt.Errorf("version not found")
}

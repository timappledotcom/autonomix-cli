package binary

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type InstallResult struct {
	Path         string
	Method       InstallMethod
	RequiredSudo bool
	InPath       bool
}

// InstallBinary installs binary to system
func InstallBinary(binaryPath, appName string, method InstallMethod) (*InstallResult, error) {
	targetPath, selectedMethod, requiresSudo := determineInstallPath(appName, method)

	if requiresSudo {
		if err := installWithSudo(binaryPath, targetPath); err != nil {
			return nil, err
		}
	} else {
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		if err := copyBinary(binaryPath, targetPath); err != nil {
			return nil, err
		}
	}

	inPath := isInPath(filepath.Dir(targetPath))

	return &InstallResult{
		Path:         targetPath,
		Method:       selectedMethod,
		RequiredSudo: requiresSudo,
		InPath:       inPath,
	}, nil
}

func determineInstallPath(appName string, method InstallMethod) (string, InstallMethod, bool) {
	home, _ := os.UserHomeDir()

	if method == Auto {
		// 1. ~/.local/bin (if exists and is in PATH)
		localBin := filepath.Join(home, ".local", "bin")
		if _, err := os.Stat(localBin); err == nil && isInPath(localBin) {
			return filepath.Join(localBin, appName), UserPath, false
		}

		// 2. /usr/local/bin (macOS)
		return filepath.Join("/usr/local/bin", appName), SystemPath, true
	}

	switch method {
	case SystemPath:
		return filepath.Join("/usr/local/bin", appName), SystemPath, true
	case UserPath:
		return filepath.Join(home, ".local", "bin", appName), UserPath, false
	case AutonomixPath:
		return filepath.Join(home, ".autonomix", "bin", appName), AutonomixPath, false
	}

	return filepath.Join(home, ".autonomix", "bin", appName), AutonomixPath, false
}

func isInPath(dir string) bool {
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, ":")
	for _, p := range paths {
		if p == dir {
			return true
		}
	}
	return false
}

func copyBinary(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, input, 0755); err != nil {
		return err
	}

	return nil
}

func installWithSudo(binaryPath, targetPath string) error {
	cmd := exec.Command("sudo", "cp", binaryPath, targetPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo install failed: %w", err)
	}

	cmd = exec.Command("sudo", "chmod", "755", targetPath)
	return cmd.Run()
}

// VerifyInstallation verifies binary is accessible
func VerifyInstallation(appName string) (string, error) {
	path, err := exec.LookPath(appName)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH", appName)
	}
	return path, nil
}

// GetInstallInstructions generates instructions for user
func GetInstallInstructions(result *InstallResult) string {
	if result.InPath {
		return fmt.Sprintf("✓ Installed at: %s\n✓ Ready to use: %s", result.Path, filepath.Base(result.Path))
	}

	dir := filepath.Dir(result.Path)
	return fmt.Sprintf("✓ Installed at: %s\n⚠ Add to your PATH:\n  export PATH=\"%s:$PATH\"", result.Path, dir)
}

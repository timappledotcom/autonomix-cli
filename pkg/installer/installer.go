package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tim/autonomix-cli/pkg/github"
	"github.com/tim/autonomix-cli/pkg/packages"
	"github.com/tim/autonomix-cli/pkg/system"
)

// DownloadUpdate finds and downloads the update, returning the path to the file.
func DownloadUpdate(release *github.Release) (string, error) {
	sysType := system.GetSystemPreferredType()
	if sysType == packages.Unknown {
		return "", fmt.Errorf("could not detect system package manager (dpkg, rpm, pacman)")
	}

	asset, err := findMatchingAsset(release.Assets, sysType)
	if err != nil {
		return "", err
	}

	// Create temp file
	tempDir := os.TempDir()
	fileName := asset.Name
	downloadPath := filepath.Join(tempDir, fileName)

	// Download
	// In a real CLI we might want progress, but for now blocking is okay or we'll assume TUI handles spinner.
	fmt.Printf("Downloading %s...\n", asset.BrowserDownloadURL)
	if err := downloadFile(downloadPath, asset.BrowserDownloadURL); err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	
	return downloadPath, nil
}

// GetInstallCmd returns the exec.Cmd to install the package.
// It does NOT set Stdin/Stdout/Stderr, the caller should do that or use tea.Exec
func GetInstallCmd(path string) (*exec.Cmd, error) {
	sysType := system.GetSystemPreferredType()
	
	switch sysType {
	case packages.Deb:
		// sudo apt-get install -y ./path
		// Using relative path for apt sometimes requires ./
		absPath, _ := filepath.Abs(path)
		return exec.Command("sudo", "apt-get", "install", "-y", absPath), nil
	case packages.Rpm:
		return exec.Command("sudo", "rpm", "-Uvh", path), nil
	case packages.Pacman:
		return exec.Command("sudo", "pacman", "-U", "--noconfirm", path), nil
	default:
		return nil, fmt.Errorf("unsupported install type: %s", sysType)
	}
}

func InstallUpdate(release *github.Release) error {
	path, err := DownloadUpdate(release)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	cmd, err := GetInstallCmd(path)
	if err != nil {
		return err
	}

	// Connect to stdout/stderr so user sees password prompt and output
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Installing %s...\n", path)
	return cmd.Run()
}

func findMatchingAsset(assets []github.Asset, sysType packages.Type) (*github.Asset, error) {
	arch := runtime.GOARCH
	// Map go arch to package arch strings commonly used
	archKeywords := []string{arch}
	if arch == "amd64" {
		archKeywords = append(archKeywords, "x86_64", "x64")
	} else if arch == "arm64" {
		archKeywords = append(archKeywords, "aarch64", "armv8")
	}

	for _, asset := range assets {
		detectedType := packages.DetectType(asset.Name)
		if detectedType != sysType {
			continue
		}

		// Check arch
		nameLower := strings.ToLower(asset.Name)
		for _, kw := range archKeywords {
			if strings.Contains(nameLower, kw) {
				return &asset, nil
			}
		}
		
		// Fallback: if no arch info is in the name, but type matches, it might be universal or the only one.
		// But risky. Let's look for one that doesn't contradict.
		// Actually, let's just return the first match of the type if strict arch match fails, 
		// but typically release assets have arch in name.
	}

	return nil, fmt.Errorf("no matching asset found for type %s and arch %s", sysType, arch)
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}


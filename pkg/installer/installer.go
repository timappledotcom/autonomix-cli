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

	"github.com/tim/autonomix-cli/pkg/binary"
	"github.com/tim/autonomix-cli/pkg/github"
	"github.com/tim/autonomix-cli/pkg/homebrew"
	"github.com/tim/autonomix-cli/pkg/packages"
	"github.com/tim/autonomix-cli/pkg/system"
)

type InstallOptions struct {
	Method      binary.InstallMethod
	ForceMethod bool
	Interactive bool
}

type InstallResult struct {
	Method  string
	Version string
	Path    string
	Success bool
	Message string
}

// GetCompatibleAssets returns a list of assets that are compatible with the current system.
func GetCompatibleAssets(release *github.Release) ([]github.Asset, error) {
	sysType := system.GetSystemPreferredType()
	if sysType == packages.Unknown {
		return nil, fmt.Errorf("could not detect system package manager")
	}

	arch := runtime.GOARCH
	// Map go arch to package arch strings commonly used
	archKeywords := []string{arch}
	if arch == "amd64" {
		archKeywords = append(archKeywords, "x86_64", "x64")
	} else if arch == "arm64" {
		archKeywords = append(archKeywords, "aarch64", "armv8")
	}
	
	// Add universal/architecture-independent keywords
	archKeywords = append(archKeywords, "all", "noarch", "any")

	var compatible []github.Asset
	availableTypes := make(map[packages.Type]bool)
	
	for _, asset := range release.Assets {
		detectedType := packages.DetectType(asset.Name)
		if detectedType != packages.Unknown {
			availableTypes[detectedType] = true
		}
		
		if detectedType != sysType {
			continue
		}

		nameLower := strings.ToLower(asset.Name)
		
		// Include if it matches arch, or if it seems universal (no arch keyword)
		// But excluding if it matches wrong arch is safer.
		// Let's include if it matches at least one keyword.
		
		matchedArch := false
		for _, kw := range archKeywords {
			if strings.Contains(nameLower, kw) {
				matchedArch = true
				break
			}
		}

		if matchedArch {
			compatible = append(compatible, asset)
		}
	}
	
	// If no strict matches, do we want to search for "noarch" or "all"?
	if len(compatible) == 0 {
		for _, asset := range release.Assets {
			detectedType := packages.DetectType(asset.Name)
			if detectedType != sysType {
				continue
			}
			// Check if it explicitly says another arch
			// If not, maybe it's universal?
			// This is heuristic.
		}
	}

	// If still no compatible assets, provide helpful error message
	if len(compatible) == 0 && len(availableTypes) > 0 {
		var typeNames []string
		for t := range availableTypes {
			typeNames = append(typeNames, string(t))
		}
		return nil, fmt.Errorf("no %s packages found for %s. Available types: %s", 
			sysType, arch, strings.Join(typeNames, ", "))
	}

	return compatible, nil
}

// GetAllAssets returns all installable assets from a release, regardless of system compatibility.
// Useful as a fallback when no compatible assets are found.
func GetAllAssets(release *github.Release) []github.Asset {
	arch := runtime.GOARCH
	archKeywords := []string{arch}
	if arch == "amd64" {
		archKeywords = append(archKeywords, "x86_64", "x64")
	} else if arch == "arm64" {
		archKeywords = append(archKeywords, "aarch64", "armv8")
	}
	archKeywords = append(archKeywords, "all", "noarch", "any")
	
	var all []github.Asset
	for _, asset := range release.Assets {
		detectedType := packages.DetectType(asset.Name)
		// Only include recognized package types
		if detectedType == packages.Unknown {
			continue
		}
		
		// Filter by arch
		nameLower := strings.ToLower(asset.Name)
		matchedArch := false
		for _, kw := range archKeywords {
			if strings.Contains(nameLower, kw) {
				matchedArch = true
				break
			}
		}
		
		if matchedArch {
			all = append(all, asset)
		}
	}
	return all
}

// DownloadAsset downloads the specified asset
func DownloadAsset(asset *github.Asset) (string, error) {
	tempDir := os.TempDir()
	fileName := asset.Name
	downloadPath := filepath.Join(tempDir, fileName)

	fmt.Printf("Downloading %s...\n", asset.BrowserDownloadURL)
	if err := downloadFile(downloadPath, asset.BrowserDownloadURL); err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	
	return downloadPath, nil
}

// DownloadUpdate finds and downloads the update, returning the path to the file.
func DownloadUpdate(release *github.Release) (string, error) {
	assets, err := GetCompatibleAssets(release)
	if err != nil {
		return "", err
	}
	if len(assets) == 0 {
		return "", fmt.Errorf("no compatible assets found")
	}
	
	// Default behavior: pick the first one
	return DownloadAsset(&assets[0])
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

func InstallUpdate(release *github.Release, opts *InstallOptions) (*InstallResult, error) {
	if opts == nil {
		opts = &InstallOptions{Method: binary.Auto}
	}

	if !opts.ForceMethod || opts.Method == binary.Auto {
		result, err := tryPackageInstall(release)
		if err == nil {
			return result, nil
		}
	}

	return tryBinaryInstall(release, opts)
}

func tryPackageInstall(release *github.Release) (*InstallResult, error) {
	path, err := DownloadUpdate(release)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)

	cmd, err := GetInstallCmd(path)
	if err != nil {
		return nil, err
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Installing %s...\n", path)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &InstallResult{
		Method:  "package",
		Version: release.TagName,
		Path:    path,
		Success: true,
		Message: "Installed via package manager",
	}, nil
}

func tryBinaryInstall(release *github.Release, opts *InstallOptions) (*InstallResult, error) {
	binaries := binary.DetectBinaryAssets(release)
	if len(binaries) == 0 {
		return nil, fmt.Errorf("no binary assets found")
	}

	selected := binaries[0]
	for _, b := range binaries {
		if b.Priority > selected.Priority {
			selected = b
		}
	}

	assetPath, err := DownloadAsset(&selected.Asset)
	if err != nil {
		return nil, err
	}
	defer os.Remove(assetPath)

	binaryPath, err := binary.ExtractBinary(assetPath, selected.BinaryName)
	if err != nil {
		return nil, err
	}
	defer os.Remove(binaryPath)

	if runtime.GOOS == "darwin" && homebrew.IsInstalled() {
		result, err := tryHomebrewInstall(release, &selected, binaryPath)
		if err == nil {
			return result, nil
		}
	}

	return installBinaryDirect(binaryPath, selected.BinaryName, opts.Method)
}

func tryHomebrewInstall(release *github.Release, asset *binary.BinaryAsset, binaryPath string) (*InstallResult, error) {
	formula, err := homebrew.SearchFormula(asset.BinaryName)
	if err != nil {
		return nil, err
	}

	// Install official formula
	if err := homebrew.InstallOfficial(formula); err != nil {
		return nil, err
	}

	return &InstallResult{
		Method:  "homebrew",
		Version: release.TagName,
		Path:    "",
		Success: true,
		Message: fmt.Sprintf("Installed %s via Homebrew", formula),
	}, nil
}

func installBinaryDirect(binaryPath, appName string, method binary.InstallMethod) (*InstallResult, error) {
	result, err := binary.InstallBinary(binaryPath, appName, method)
	if err != nil {
		return nil, err
	}

	return &InstallResult{
		Method:  "binary",
		Version: "",
		Path:    result.Path,
		Success: true,
		Message: binary.GetInstallInstructions(result),
	}, nil
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

	// Add universal/architecture-independent keywords
	archKeywords = append(archKeywords, "all", "noarch", "any")

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


// InstallAndUpdateConfig installs app and updates config
func InstallAndUpdateConfig(cfg interface{}, appIndex int, opts *InstallOptions) (*InstallResult, error) {
	// This function requires access to config.Config and config.App
	// For now we return an error indicating it must be implemented in the caller
	return nil, fmt.Errorf("InstallAndUpdateConfig must be implemented in the package that has access to config")
}

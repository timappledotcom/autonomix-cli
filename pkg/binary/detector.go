package binary

import (
	"runtime"
	"strings"

	"github.com/tim/autonomix-cli/pkg/github"
)

type InstallMethod int

const (
	Auto InstallMethod = iota
	SystemPath
	UserPath
	Homebrew
	AutonomixPath
)

type BinaryAsset struct {
	Asset      github.Asset
	BinaryName string
	IsArchive  bool
	Priority   int
}

// DetectBinaryAssets finds binary assets compatible with current platform
func DetectBinaryAssets(release *github.Release) []BinaryAsset {
	var binaries []BinaryAsset
	
	for _, asset := range release.Assets {
		if !IsBinaryAsset(asset) {
			continue
		}
		
		if !MatchesPlatform(asset.Name) {
			continue
		}
		
		binary := BinaryAsset{
			Asset:      asset,
			BinaryName: GetBinaryName(asset),
			IsArchive:  isArchive(asset.Name),
			Priority:   getPriority(asset.Name),
		}
		
		binaries = append(binaries, binary)
	}
	
	return binaries
}

// IsBinaryAsset checks if asset is an executable binary
func IsBinaryAsset(asset github.Asset) bool {
	name := strings.ToLower(asset.Name)
	
	if strings.Contains(name, "checksum") || strings.Contains(name, "sha256") ||
		strings.Contains(name, "sha512") || strings.HasSuffix(name, ".sig") ||
		strings.HasSuffix(name, ".asc") {
		return false
	}
	
	if strings.HasPrefix(name, "source") || strings.Contains(name, "src") {
		return false
	}
	
	return !strings.HasSuffix(name, ".deb") &&
		!strings.HasSuffix(name, ".rpm") &&
		!strings.HasSuffix(name, ".apk") &&
		!strings.HasSuffix(name, ".dmg") &&
		!strings.HasSuffix(name, ".pkg")
}

// GetBinaryName extracts binary name from asset
func GetBinaryName(asset github.Asset) string {
	name := asset.Name
	
	name = strings.TrimSuffix(name, ".tar.gz")
	name = strings.TrimSuffix(name, ".tgz")
	name = strings.TrimSuffix(name, ".zip")
	name = strings.TrimSuffix(name, ".gz")
	
	parts := strings.Split(name, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	
	return name
}

// MatchesPlatform checks if asset is compatible with current OS and architecture
func MatchesPlatform(assetName string) bool {
	name := strings.ToLower(assetName)
	
	osMatch := false
	switch runtime.GOOS {
	case "darwin":
		osMatch = strings.Contains(name, "darwin") || strings.Contains(name, "macos") || strings.Contains(name, "osx")
	case "linux":
		osMatch = strings.Contains(name, "linux")
	}
	
	if !osMatch {
		return false
	}
	
	archMatch := false
	switch runtime.GOARCH {
	case "amd64":
		archMatch = strings.Contains(name, "amd64") || strings.Contains(name, "x86_64") || strings.Contains(name, "x64")
	case "arm64":
		archMatch = strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")
	}
	
	return archMatch
}

func isArchive(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".gz")
}

func getPriority(name string) int {
	lower := strings.ToLower(name)
	
	// Standalone binary (highest priority)
	if !isArchive(name) {
		return 3
	}
	
	// .tar.gz
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return 2
	}
	
	// .zip
	if strings.HasSuffix(lower, ".zip") {
		return 1
	}
	
	return 0
}

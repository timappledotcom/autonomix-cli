package system

import (
	"os/exec"
	"strings"

	"github.com/tim/autonomix-cli/pkg/packages"
)

// CheckInstalled checks if an application is installed via various package managers.
// It returns the version string, the package type, true if found.
func CheckInstalled(appName string) (string, packages.Type, bool) {
	// Generate candidate names to check
	// e.g. "My App" -> ["My App", "my app", "my-app"]
	candidates := []string{appName}
	
	lower := strings.ToLower(appName)
	if lower != appName {
		candidates = append(candidates, lower)
	}
	
	kebab := strings.ReplaceAll(lower, " ", "-")
	if kebab != lower && kebab != appName {
		candidates = append(candidates, kebab)
	}

	for _, name := range candidates {
		if name == "" {
			continue
		}
		
		// Try each package manager with this name
		
		// Check Snap
		if ver, ok := checkSnap(name); ok {
			return ver, packages.Snap, true
		}
		
		// Check Flatpak
		if ver, ok := checkFlatpak(name); ok {
			return ver, packages.Flatpak, true
		}
		
		// Check Dpkg (Debian/Ubuntu)
		if ver, ok := checkDpkg(name); ok {
			return ver, packages.Deb, true
		}
		
		// Check Pacman (Arch)
		if ver, ok := checkPacman(name); ok {
			return ver, packages.Pacman, true
		}
		
		// Check RPM
		if ver, ok := checkRpm(name); ok {
			return ver, packages.Rpm, true
		}
	}

	return "", packages.Unknown, false
}

// GetSystemPreferredType returns the preferred package type for the running system
func GetSystemPreferredType() packages.Type {
	if _, err := exec.LookPath("dpkg"); err == nil {
		return packages.Deb
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return packages.Pacman
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		return packages.Rpm
	}
	return packages.Unknown
}


func checkSnap(name string) (string, bool) {
	// snap list name
	cmd := exec.Command("snap", "list", name)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return "", false
	}
	fields := strings.Fields(lines[1])
	if len(fields) >= 2 {
		return fields[1], true
	}
	return "", false
}

func checkFlatpak(name string) (string, bool) {
	// flatpak list --app --columns=application,version
	cmd := exec.Command("flatpak", "list", "--app", "--columns=application,name,version")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	
	lowerName := strings.ToLower(name)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// Expected: com.example.App Name Version
		if len(fields) >= 3 {
			appID := strings.ToLower(fields[0])
			appName := strings.ToLower(fields[1])
			
			// Heuristic: if ID ends with name or name matches
			if appName == lowerName || strings.HasSuffix(appID, "." + lowerName) {
				return fields[2], true
			}
		}
	}
	return "", false
}

func checkDpkg(name string) (string, bool) {
	// dpkg-query -W -f='${Version}' name
	cmd := exec.Command("dpkg-query", "-W", "-f=${Version}", name)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return string(out), true
	}
	return "", false
}

func checkPacman(name string) (string, bool) {
	// pacman -Q name
    // output: name version
	cmd := exec.Command("pacman", "-Q", name)
	out, err := cmd.Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 2 {
			return parts[1], true
		}
	}
	return "", false
}

func checkRpm(name string) (string, bool) {
	// rpm -q --qf "%{VERSION}" name
	cmd := exec.Command("rpm", "-q", "--qf", "%{VERSION}", name)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return string(out), true
	}
	return "", false
}

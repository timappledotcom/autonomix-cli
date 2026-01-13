package system

import (
	"os/exec"
	"strings"
)

// CheckInstalled checks if an application is installed via various package managers.
// It returns the version string, true if found, and any error encountered (rarely used).
func CheckInstalled(appName string) (string, bool) {
	// Try each package manager
	
	// Check Snap
	if ver, ok := checkSnap(appName); ok {
		return ver, true
	}
	
	// Check Flatpak
	// Flatpak naming is usually reverse DNS (com.example.App), so simple name match is hard.
	// We'll try a search-like approach on the list.
	if ver, ok := checkFlatpak(appName); ok {
		return ver, true
	}
	
	// Check Dpkg (Debian/Ubuntu)
	if ver, ok := checkDpkg(appName); ok {
		return ver, true
	}
	
	// Check Pacman (Arch)
	if ver, ok := checkPacman(appName); ok {
		return ver, true
	}
	
	// Check RPM
	if ver, ok := checkRpm(appName); ok {
		return ver, true
	}

	return "", false
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

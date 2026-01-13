package packages

import (
	"strings"
)

type Type string

const (
	Deb     Type = "deb"
	Rpm     Type = "rpm"
	Flatpak Type = "flatpak"
	Snap    Type = "snap"
	Pacman  Type = "pacman"
	AppImage Type = "appimage"
	Unknown Type = "unknown"
)

func DetectType(filename string) Type {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".deb") {
		return Deb
	}
	if strings.HasSuffix(lower, ".rpm") {
		return Rpm
	}
	if strings.HasSuffix(lower, ".flatpak") || strings.HasSuffix(lower, ".flatpakref") {
		return Flatpak
	}
	if strings.HasSuffix(lower, ".snap") {
		return Snap
	}
	if strings.HasSuffix(lower, ".pkg.tar.zst") || strings.HasSuffix(lower, ".pkg.tar.xz") {
		return Pacman
	}
	if strings.HasSuffix(lower, ".appimage") {
		return AppImage // Bonus, usually useful
	}
	return Unknown
}

func DisplayName(t Type) string {
	switch t {
	case Deb:
		return "Debian Package (.deb)"
	case Rpm:
		return "RPM Package (.rpm)"
	case Flatpak:
		return "Flatpak"
	case Snap:
		return "Snap Package"
	case Pacman:
		return "Arch Package"
	case AppImage:
		return "AppImage"
	default:
		return "Unknown"
	}
}

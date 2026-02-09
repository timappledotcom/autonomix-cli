# Autonomix CLI v0.3

Autonomix CLI is a terminal-based utility written in Go that allows you to easily install and manage applications directly from GitHub Releases.

## Features

- **Install from GitHub**: Add any GitHub repository URL to track.
- **Multiple Install Methods**: Supports system packages (`.deb`, `.rpm`, `.flatpak`, `.snap`, `.appimage`, Arch packages), Homebrew (macOS), and direct binary installation.
- **Smart Updates**: Checks for new releases on GitHub.
- **System Integration**: Detects if the application is already installed on your system and shows the installed version.
- **CLI & TUI**: Full command-line interface with interactive Terminal User Interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Installation

### From GitHub Releases

Go to the [Releases](https://github.com/tim/autonomix-cli/releases) page and download the package for your system.

**Debian/Ubuntu:**
```bash
sudo dpkg -i autonomix-cli_*.deb
```

**Fedora/RHEL:**
```bash
sudo rpm -i autonomix-cli_*.rpm
```

**Arch Linux:**
```bash
sudo pacman -U autonomix-cli_*.pkg.tar.zst
```

## Usage

### Interactive TUI

Run without arguments to launch the interactive interface:

```bash
autonomix-cli
```

**TUI Controls:**
- **Start Typing**: To add a new GitHub repository URL.
- **Enter**: Confirm adding a repo.
- **u**: Check for updates for the selected app.
- **d**: Delete/Remove an app from the list (stops tracking).
- **q / Ctrl+C**: Quit.

### Command Line Interface

```bash
autonomix-cli add <github-url>       # Add repository (auto-detects install method)
autonomix-cli add --brew <url>       # Force Homebrew (macOS)
autonomix-cli add --binary <url>     # Force direct binary
autonomix-cli add --system <url>     # Force system package
autonomix-cli list                   # List tracked apps
autonomix-cli update <app-name>      # Update an app
autonomix-cli remove <app-name>      # Remove an app
autonomix-cli clean                  # Remove untracked apps
autonomix-cli --version              # Show version
```

## Configuration

Configuration is stored in `~/.autonomix/config.json`.

## building

```bash
go build
```

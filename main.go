package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tim/autonomix-cli/config"
	"github.com/tim/autonomix-cli/pkg/cli"
	"github.com/tim/autonomix-cli/tui"
)

const SelfRepoURL = "https://github.com/timappledotcom/autonomix-cli"

var version = "v0.3.0"

func main() {
	if len(os.Args) > 1 {
		cli.HandleCommand(os.Args[1:], version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Ensure self is tracked
	tracked := false
	for i, app := range cfg.Apps {
		if app.RepoURL == SelfRepoURL {
			tracked = true
			if app.Version != version {
				cfg.Apps[i].Version = version
			}
			break
		}
	}
	if !tracked {
		cfg.Apps = append(cfg.Apps, config.App{
			Name:    "autonomix-cli",
			RepoURL: SelfRepoURL,
			Version: version,
		})
	}
	config.Save(cfg)

	p := tea.NewProgram(tui.NewModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

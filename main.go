package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tim/autonomix-cli/config"
	"github.com/tim/autonomix-cli/tui"
)

const SelfRepoURL = "https://github.com/tim/autonomix-cli"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Ensure self is tracked
	tracked := false
	for _, app := range cfg.Apps {
		if app.RepoURL == SelfRepoURL {
			tracked = true
			break
		}
	}
	if !tracked {
		cfg.Apps = append(cfg.Apps, config.App{
			Name:    "Autonomix CLI",
			RepoURL: SelfRepoURL,
			Version: "dev", // Initial version
		})
		config.Save(cfg)
	}

	p := tea.NewProgram(tui.NewModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/tim/autonomix-cli/config"
	"github.com/tim/autonomix-cli/pkg/binary"
	"github.com/tim/autonomix-cli/pkg/github"
	"github.com/tim/autonomix-cli/pkg/manager"
)

func HandleCommand(args []string, version string) {
	if len(args) == 0 {
		return
	}

	cmd := args[0]
	switch cmd {
	case "add":
		handleAdd(args[1:])
	case "update":
		handleUpdate(args[1:])
	case "list":
		handleList()
	case "remove":
		handleRemove(args[1:])
	case "clean":
		handleClean()
	case "--help", "-h":
		printHelp(version)
	case "--version", "-v":
		fmt.Printf("autonomix-cli %s\n", version)
	}
}

func handleAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	brew := fs.Bool("brew", false, "Force Homebrew")
	binaryFlag := fs.Bool("binary", false, "Force binary")
	system := fs.Bool("system", false, "System path")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Error: URL required")
		os.Exit(1)
	}

	method := binary.Auto
	if *brew {
		method = binary.Homebrew
	} else if *binaryFlag {
		method = binary.UserPath
	} else if *system {
		method = binary.SystemPath
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Adding %s...\n", fs.Arg(0))
	res, err := manager.AddApp(cfg, fs.Arg(0))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Tracked %s (Latest: %s)\n", res.App.Name, res.App.Latest)
	if res.App.Version != "" {
		fmt.Printf("  Already installed: %s\n", res.App.Version)
		app := &cfg.Apps[len(cfg.Apps)-1]
		app.InstallStatus = config.StatusInstalled
		config.Save(cfg)
		return
	}

	// Now install
	fmt.Printf("Installing...\n")
	rel, err := github.GetLatestRelease(res.App.RepoURL)
	if err != nil {
		fmt.Printf("Error fetching release: %v\n", err)
		os.Exit(1)
	}

	app := &cfg.Apps[len(cfg.Apps)-1]
	if err := manager.InstallApp(rel, app, method); err != nil {
		config.Save(cfg)
		fmt.Printf("Error installing: %v\n", err)
		os.Exit(1)
	}

	config.Save(cfg)
	fmt.Printf("✓ Installed %s\n", app.Version)
	if app.BinaryPath != "" {
		fmt.Printf("  Path: %s\n", app.BinaryPath)
	}
}

func handleUpdate(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: app name required")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for i, app := range cfg.Apps {
		if app.Name == args[0] {
			fmt.Printf("Updating %s...\n", args[0])
			cfg.Apps[i].Version = app.Latest
			config.Save(cfg)
			fmt.Printf("✓ Updated to %s\n", app.Latest)
			return
		}
	}

	fmt.Printf("Error: %s not found\n", args[0])
	os.Exit(1)
}

func handleList() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Apps) == 0 {
		fmt.Println("No apps tracked")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tLATEST\tMETHOD\tSTATUS")
	for _, app := range cfg.Apps {
		method := app.InstallMethod
		if method == "" {
			method = "-"
		}
		
		status := "-"
		if app.InstallStatus == config.StatusInstalled {
			status = "✓ Installed"
		} else if app.InstallStatus == config.StatusFailed {
			status = "✗ " + app.InstallError
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", app.Name, app.Version, app.Latest, method, status)
	}
	w.Flush()
}

func handleClean() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	removed := 0
	filtered := []config.App{}
	for _, app := range cfg.Apps {
		if app.InstallStatus == config.StatusFailed {
			fmt.Printf("Removing failed: %s\n", app.Name)
			removed++
		} else {
			filtered = append(filtered, app)
		}
	}

	if removed == 0 {
		fmt.Println("No failed installations to clean")
		return
	}

	cfg.Apps = filtered
	config.Save(cfg)
	fmt.Printf("✓ Cleaned %d failed installation(s)\n", removed)
}

func handleRemove(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: app name required")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for i, app := range cfg.Apps {
		if app.Name == args[0] {
			uninstallApp(&app)
			cfg.Apps = append(cfg.Apps[:i], cfg.Apps[i+1:]...)
			config.Save(cfg)
			fmt.Printf("✓ Removed %s\n", args[0])
			return
		}
	}

	fmt.Printf("Error: %s not found\n", args[0])
	os.Exit(1)
}

func uninstallApp(app *config.App) {
	switch app.InstallMethod {
	case config.InstallMethodHomebrew:
		fmt.Printf("Uninstalling via Homebrew...\n")
		cmd := exec.Command("brew", "uninstall", app.Name)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: brew uninstall failed: %v\n", err)
		}
	case config.InstallMethodBinary:
		if app.BinaryPath != "" {
			fmt.Printf("Removing binary: %s\n", app.BinaryPath)
			if err := os.Remove(app.BinaryPath); err != nil {
				fmt.Printf("Warning: failed to remove: %v\n", err)
			}
		}
	}
}

func printHelp(version string) {
	fmt.Printf(`autonomix-cli %s

USAGE:
  autonomix-cli              Launch TUI
  autonomix-cli add <url>    Add repository
  autonomix-cli update <app> Update app
  autonomix-cli list         List tracked apps
  autonomix-cli remove <app> Remove app
  autonomix-cli clean        Remove failed installations

FLAGS (add):
  --brew    Homebrew
  --binary  Binary install
  --system  System path

OPTIONS:
  -h, --help     Show help
  -v, --version  Show version
`, version)
}

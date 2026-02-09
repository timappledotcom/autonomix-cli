package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	InstallMethodPackage  = "package"
	InstallMethodBinary   = "binary"
	InstallMethodHomebrew = "homebrew"
	InstallMethodUnknown  = ""

	StatusInstalled = "installed"
	StatusFailed    = "failed"
)

type App struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Version     string `json:"version"` // Installed version
	Latest      string `json:"latest"`  // Latest version detected
	LastChecked string `json:"last_checked"`

	InstallMethod string `json:"install_method,omitempty"`
	BinaryPath    string `json:"binary_path,omitempty"`
	InstallStatus string `json:"install_status,omitempty"`
	InstallError  string `json:"install_error,omitempty"`
}

type Config struct {
	Apps []App `json:"apps"`
}



func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".autonomix"), nil
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{Apps: []App{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func detectInstallMethod(app *App) string {
	path, err := exec.LookPath(app.Name)
	if err != nil {
		return InstallMethodUnknown
	}

	app.BinaryPath = path

	if strings.Contains(path, "/Cellar/") || strings.Contains(path, "/homebrew/") {
		cmd := exec.Command("brew", "list", app.Name)
		if cmd.Run() == nil {
			return InstallMethodHomebrew
		}
	}

	return InstallMethodBinary
}

func Save(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Migrate on save if needed
	for i := range cfg.Apps {
		if cfg.Apps[i].InstallMethod == "" && cfg.Apps[i].Version != "" {
			cfg.Apps[i].InstallMethod = detectInstallMethod(&cfg.Apps[i])
		}
	}

	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

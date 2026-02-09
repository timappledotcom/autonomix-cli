package manager

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/tim/autonomix-cli/config"
	"github.com/tim/autonomix-cli/pkg/binary"
	"github.com/tim/autonomix-cli/pkg/github"
	"github.com/tim/autonomix-cli/pkg/homebrew"
	"github.com/tim/autonomix-cli/pkg/installer"
	"github.com/tim/autonomix-cli/pkg/system"
)

type AddResult struct {
	App     config.App
	Created bool
}

func AddApp(cfg *config.Config, repoURL string) (*AddResult, error) {
	repoURL = cleanRepoURL(repoURL)

	for _, app := range cfg.Apps {
		if strings.EqualFold(app.RepoURL, repoURL) {
			return &AddResult{App: app, Created: false}, fmt.Errorf("repository already tracked")
		}
	}

	rel, err := github.GetLatestRelease(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}

	appName := getAppName(repoURL, rel)
	repoName := getRepoName(repoURL)

	newApp := config.App{
		Name:    appName,
		RepoURL: repoURL,
		Latest:  rel.TagName,
	}

	if ver, _, installed := system.CheckInstalled(appName); installed {
		newApp.Version = ver
	} else if repoName != "" && repoName != appName {
		if ver, _, installed := system.CheckInstalled(repoName); installed {
			newApp.Version = ver
		}
	}

	cfg.Apps = append(cfg.Apps, newApp)
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &AddResult{App: newApp, Created: true}, nil
}

func cleanRepoURL(url string) string {
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) == 2 {
			pathParts := strings.Split(strings.Trim(parts[1], "/"), "/")
			if len(pathParts) >= 2 {
				return "https://github.com/" + pathParts[0] + "/" + pathParts[1]
			}
		}
	}
	return url
}

func getAppName(repoURL string, rel *github.Release) string {
	repoName := getRepoName(repoURL)

	if rel.Name != "" && !strings.HasPrefix(rel.Name, "v") &&
		!strings.Contains(strings.ToLower(rel.Name), "release") &&
		rel.Name != rel.TagName {
		return rel.Name
	}
	return repoName
}

func getRepoName(repoURL string) string {
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func InstallApp(rel *github.Release, app *config.App, method binary.InstallMethod) error {
	if method != binary.Auto {
		return installWithMethod(rel, app, method)
	}

	// Auto: try package, homebrew, then binary
	if err := tryPackageInstall(rel, app); err == nil {
		app.InstallStatus = config.StatusInstalled
		return nil
	}

	if runtime.GOOS == "darwin" {
		if err := tryHomebrewInstall(app); err == nil {
			app.InstallStatus = config.StatusInstalled
			return nil
		}
	}

	if err := tryBinaryInstall(rel, app, binary.Auto); err != nil {
		app.InstallStatus = config.StatusFailed
		app.InstallError = err.Error()
		return err
	}

	app.InstallStatus = config.StatusInstalled
	return nil
}

func installWithMethod(rel *github.Release, app *config.App, method binary.InstallMethod) error {
	if method == binary.Homebrew {
		if err := tryHomebrewInstall(app); err != nil {
			app.InstallStatus = config.StatusFailed
			app.InstallError = err.Error()
			return err
		}
		app.InstallStatus = config.StatusInstalled
		return nil
	}
	
	if err := tryBinaryInstall(rel, app, method); err != nil {
		app.InstallStatus = config.StatusFailed
		app.InstallError = err.Error()
		return err
	}
	app.InstallStatus = config.StatusInstalled
	return nil
}

func tryPackageInstall(rel *github.Release, app *config.App) error {
	assets, err := installer.GetCompatibleAssets(rel)
	if err != nil || len(assets) == 0 {
		return fmt.Errorf("no compatible assets")
	}

	if _, err := installer.InstallUpdate(rel, &installer.InstallOptions{Method: binary.Auto}); err != nil {
		return err
	}

	app.Version = strings.TrimPrefix(rel.TagName, "v")
	app.InstallMethod = config.InstallMethodPackage
	return nil
}

func tryHomebrewInstall(app *config.App) error {
	if !homebrew.IsInstalled() {
		return fmt.Errorf("homebrew not installed")
	}

	formula, err := homebrew.SearchFormula(app.Name)
	if err != nil {
		return err
	}

	if err := homebrew.InstallOfficial(formula); err != nil {
		return err
	}

	if ver, err := homebrew.GetInstalledVersion(app.Name); err == nil {
		app.Version = ver
	}
	app.InstallMethod = config.InstallMethodHomebrew
	return nil
}

func tryBinaryInstall(rel *github.Release, app *config.App, method binary.InstallMethod) error {
	result, err := installer.InstallUpdate(rel, &installer.InstallOptions{Method: method})
	if err != nil {
		return err
	}

	app.Version = strings.TrimPrefix(rel.TagName, "v")
	app.BinaryPath = result.Path
	app.InstallMethod = config.InstallMethodBinary
	return nil
}

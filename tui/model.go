package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tim/autonomix-cli/config"
	"github.com/tim/autonomix-cli/pkg/github"
	"github.com/tim/autonomix-cli/pkg/system"
)

var (
	docStyle         = lipgloss.NewStyle().Margin(1, 2)
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	installedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")) // Green
	updateStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	notInstalledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")) // Grey
)

type state int

const (
	viewList state = iota
	viewAdd
)

type item struct {
	app config.App
}

func (i item) Title() string       { return i.app.Name }
func (i item) Description() string {
	status := "Not Installed"
	style := notInstalledStyle
	
	if i.app.Version != "" {
		status = "Installed: " + i.app.Version
		style = installedStyle
		if i.app.Latest != "" && i.app.Latest != i.app.Version {
			status = fmt.Sprintf("Update Available: %s -> %s", i.app.Version, i.app.Latest)
			style = updateStyle
		}
	}
	
	return fmt.Sprintf("%s (%s)", i.app.RepoURL, style.Render(status))
}
func (i item) FilterValue() string { return i.app.Name }

type Model struct {
	list      list.Model
	input     textinput.Model
	state     state
	config    *config.Config
	quitting  bool
	err       error
}

func NewModel(cfg *config.Config) Model {
	items := []list.Item{}
	for _, app := range cfg.Apps {
		items = append(items, item{app: app})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Autonomix Apps"

	ti := textinput.New()
	ti.Placeholder = "https://github.com/owner/repo"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return Model{
		list:   l,
		input:  ti,
		state:  viewList,
		config: cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == viewAdd {
			switch msg.Type {
			case tea.KeyEnter:
				url := m.input.Value()
				if url != "" {
					return m, checkRepoArgCmd(url)
				}
				m.state = viewList
				m.input.Reset()
				return m, nil
			case tea.KeyEsc:
				m.state = viewList
				m.input.Reset()
				return m, nil
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		if m.state == viewList {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "a":
				m.state = viewAdd
				m.input.Focus()
				return m, textinput.Blink
			case "d":
				if index := m.list.Index(); index >= 0 && index < len(m.list.Items()) {
					m.config.Apps = append(m.config.Apps[:index], m.config.Apps[index+1:]...)
					config.Save(m.config) // Save immediately for now
					m.list.RemoveItem(index)
				}
				return m, nil
			case "u":
				// Check for updates for the selected item
				if index := m.list.Index(); index >= 0 && index < len(m.list.Items()) {
					selectedItem := m.list.Items()[index].(item)
					return m, checkUpdateCmd(selectedItem.app, index)
				}
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case repoCheckedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = viewList
			return m, nil // potentially show error
		}
		
		// Determine a good name for the app
		// Prefer repo name as it is most likely the package name
		parts := strings.Split(msg.repoURL, "/")
		repoName := ""
		if len(parts) > 0 {
			repoName = parts[len(parts)-1]
		}

		// Use Release Name only if it looks like a real name, otherwise repo name
		appName := msg.release.Name
		if appName == "" || strings.HasPrefix(appName, "v") || strings.Contains(strings.ToLower(appName), "release") {
			appName = repoName
		}
		
		newApp := config.App{
			Name:    appName,
			RepoURL: msg.repoURL,
			Latest:  msg.release.TagName,
		}

		// Check if installed locally
		// Try the determined app name, then fallback to repoName
		if ver, installed := system.CheckInstalled(newApp.Name); installed {
			newApp.Version = ver
		} else if repoName != "" && repoName != newApp.Name {
			if ver, installed := system.CheckInstalled(repoName); installed {
				newApp.Version = ver
			}
		}

		m.config.Apps = append(m.config.Apps, newApp)
		config.Save(m.config)
		m.list.InsertItem(len(m.list.Items()), item{app: newApp})
		m.state = viewList
		m.input.Reset()
		return m, nil

	case updateCheckedMsg:
		if msg.err != nil {
			// handle error, maybe statusbar
			return m, nil 
		}
		// update the item in the list
		idx := msg.index
		if idx >= 0 && idx < len(m.config.Apps) {
			m.config.Apps[idx].Latest = msg.release.TagName
			config.Save(m.config)
			// Update list item
			cmd = m.list.SetItem(idx, item{app: m.config.Apps[idx]})
			cmds = append(cmds, cmd)
		}
	}

	if m.state == viewList {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.state == viewAdd {
		return fmt.Sprintf(
			"Enter GitHub Repo URL:\n\n%s\n\n(esc to cancel)\n",
			m.input.View(),
		)
	}
	return docStyle.Render(m.list.View())
}

// Commands and Messages

type repoCheckedMsg struct {
	repoURL string
	release *github.Release
	err     error
}

func checkRepoArgCmd(url string) tea.Cmd {
	return func() tea.Msg {
		rel, err := github.GetLatestRelease(url)
		return repoCheckedMsg{repoURL: url, release: rel, err: err}
	}
}

type updateCheckedMsg struct {
	index   int
	release *github.Release
	err     error
}

func checkUpdateCmd(app config.App, index int) tea.Cmd {
	return func() tea.Msg {
		rel, err := github.GetLatestRelease(app.RepoURL)
		return updateCheckedMsg{index: index, release: rel, err: err}
	}
}

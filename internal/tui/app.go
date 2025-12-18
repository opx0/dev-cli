package tui

import (
	"context"
	"database/sql"
	"os"

	"dev-cli/internal/config"
	"dev-cli/internal/infra"
	"dev-cli/internal/llm"
	"dev-cli/internal/storage"
	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/tabs/assist"
	"dev-cli/internal/tui/tabs/dashboard"
	"dev-cli/internal/tui/tabs/history"
	"dev-cli/internal/tui/tabs/monitor"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SessionState int

const (
	StateLoading SessionState = iota
	StateMain
)

type AppMode int

const (
	ModeNormal AppMode = iota
	ModeInsert
)

type Tab int

const (
	TabDashboard Tab = iota
	TabMonitor
	TabAssist
	TabHistory
)

type Model struct {
	state     SessionState
	mode      AppMode
	activeTab Tab
	width     int
	height    int
	quitting  bool

	dashboard dashboard.Model
	monitor   monitor.Model
	assist    assist.Model
	history   history.Model

	tabBar    components.TabBar
	statusBar components.StatusBar
	spinner   spinner.Model
	help      help.Model

	db       *sql.DB
	aiClient *llm.HybridClient
	cwd      string
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	cwd, _ := os.Getwd()
	aiClient := llm.NewHybridClient()

	aiMode := "local"
	if config.Current.IsWebSearchEnabled() {
		aiMode = "cloud"
	}

	tabBar := components.NewTabBar([]string{
		"[1] ⊞ Dashboard",
		"[2] ~ Monitor",
		"[3] ? Assist",
		"[4] ↺ History",
	})

	return Model{
		state:     StateLoading,
		mode:      ModeNormal,
		activeTab: TabDashboard,
		cwd:       cwd,
		aiClient:  aiClient,

		dashboard: dashboard.New().SetCwd(cwd),
		monitor:   monitor.New(),
		assist:    assist.New(aiClient, aiMode),
		history:   history.New(),

		tabBar:    tabBar,
		statusBar: components.NewStatusBar(),
		spinner:   s,
		help:      help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerHealth,
		checkGPUStats,
		checkServices,
		checkDBAndHistory,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tabBar = m.tabBar.SetWidth(msg.Width)
		m.statusBar = m.statusBar.SetWidth(msg.Width)

		m.dashboard = m.dashboard.SetSize(msg.Width, msg.Height-4)
		m.monitor = m.monitor.SetSize(msg.Width, msg.Height-4)
		m.assist = m.assist.SetSize(msg.Width, msg.Height-4)
		m.history = m.history.SetSize(msg.Width, msg.Height-4)

	case dockerHealthMsg:
		m.dashboard = m.dashboard.SetDockerHealth(msg.health)
		m.monitor = m.monitor.SetDockerHealth(msg.health)
		if msg.health.Available {
			m.state = StateMain
		}

	case gpuStatsMsg:
		m.dashboard = m.dashboard.SetGPUStats(msg.stats)

	case serviceHealthMsg:
		m.dashboard = m.dashboard.SetServiceHealth(msg.services)

	case historyLoadedMsg:
		if msg.err == nil {
			m.db = msg.db
			m.history = m.history.SetHistory(msg.history)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd, checkGPUStats, checkDockerHealth, checkServices)

	case commandOutputMsg:
		m.dashboard = m.dashboard.AppendOutput(string(msg))

	case clearViewportMsg:
		m.dashboard = m.dashboard.ClearViewport()

	case tea.KeyMsg:

		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if m.mode == ModeNormal {
			switch msg.String() {
			case "1":
				m.activeTab = TabDashboard
			case "2":
				m.activeTab = TabMonitor
			case "3":
				m.activeTab = TabAssist
			case "4":
				m.activeTab = TabHistory
			case "q":
				m.quitting = true
				return m, tea.Quit
			}
		}

		var cmd tea.Cmd
		switch m.activeTab {
		case TabDashboard:
			m.dashboard, cmd = m.dashboard.Update(msg, dashboard.DefaultKeyMap())
			m.mode = m.getModeFromTab()
			cmds = append(cmds, cmd)

		case TabMonitor:
			m.monitor, cmd = m.monitor.Update(msg, monitor.DefaultKeyMap())
			cmds = append(cmds, cmd)

		case TabAssist:
			m.assist, cmd = m.assist.Update(msg, assist.DefaultKeyMap())
			m.mode = m.getModeFromTab()
			cmds = append(cmds, cmd)

		case TabHistory:
			m.history, cmd = m.history.Update(msg, history.DefaultKeyMap())
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) getModeFromTab() AppMode {
	switch m.activeTab {
	case TabDashboard:
		if m.dashboard.InsertMode() {
			return ModeInsert
		}
	case TabAssist:
		if m.assist.InsertMode() {
			return ModeInsert
		}
	}
	return ModeNormal
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.state == StateLoading {
		return m.viewLoading()
	}

	return m.viewMain()
}

func (m Model) viewLoading() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111b")).
		Background(lipgloss.Color("#cba6f7")).
		Padding(0, 1).
		Render("dev-cli")

	status := m.spinner.View() + " Checking Docker daemon..."
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585b70")).
		Padding(1, 2).
		Render(status)

	return "\n" + title + "\n\n" + box + "\n"
}

func (m Model) viewMain() string {

	m.tabBar = m.tabBar.SetActive(int(m.activeTab)).SetInsertMode(m.mode == ModeInsert)
	tabBar := m.tabBar.Render()

	var content string
	switch m.activeTab {
	case TabDashboard:
		content = m.dashboard.View()
	case TabMonitor:
		content = m.monitor.View()
	case TabAssist:
		content = m.assist.View()
	case TabHistory:
		content = m.history.View()
	}

	contentHeight := m.height - 3
	if contentHeight < 10 {
		contentHeight = 10
	}
	styledContent := lipgloss.NewStyle().Height(contentHeight).MaxWidth(m.width).Render(content)

	focusLabel := m.getFocusLabel()
	var statusBar string
	switch m.activeTab {
	case TabDashboard:
		statusBar = m.statusBar.Render(DashboardKeys, focusLabel)
	case TabMonitor:
		statusBar = m.statusBar.Render(MonitorKeys, focusLabel)
	case TabAssist:
		statusBar = m.statusBar.Render(AssistKeys, focusLabel)
	case TabHistory:
		statusBar = m.statusBar.Render(HistoryKeys, focusLabel)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, styledContent, statusBar)
}

func (m Model) getFocusLabel() string {
	switch m.activeTab {
	case TabDashboard:
		if m.dashboard.Focus() == dashboard.FocusSidebar {
			return "sidebar"
		}
		return "main"
	case TabMonitor:
		if m.monitor.Focus() == monitor.FocusSidebar {
			return "sidebar"
		}
		return "main"
	case TabAssist:
		if m.assist.Focus() == assist.FocusSidebar {
			return "sidebar"
		}
		return "main"
	case TabHistory:
		if m.history.Focus() == history.FocusSidebar {
			return "sidebar"
		}
		return "main"
	}
	return "main"
}

func checkDockerHealth() tea.Msg {
	dockerClient, err := infra.NewDockerClient()
	if err != nil {
		return dockerHealthMsg{
			health: infra.DockerHealth{
				Available: false,
				Error:     err,
			},
		}
	}
	defer dockerClient.Close()

	health := dockerClient.CheckHealth(context.Background())
	return dockerHealthMsg{health: health}
}

func checkGPUStats() tea.Msg {
	stats := infra.GetGPUStats()
	return gpuStatsMsg{stats: stats}
}

func checkServices() tea.Msg {
	services := infra.CheckServices()
	return serviceHealthMsg{services: services}
}

func checkDBAndHistory() tea.Msg {
	db, err := storage.InitDB()
	if err != nil {
		return historyLoadedMsg{err: err}
	}

	history, err := storage.GetRecentHistory(db, 50)
	if err != nil {
		return historyLoadedMsg{db: db, err: err}
	}

	return historyLoadedMsg{db: db, history: history}
}

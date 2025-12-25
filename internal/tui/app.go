package tui

import (
	"context"
	"database/sql"
	"os"

	"dev-cli/internal/infra"
	"dev-cli/internal/llm"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/plugins/ai"
	"dev-cli/internal/plugins/command"
	"dev-cli/internal/storage"
	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/tabs/agent"
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
	TabAgent Tab = iota
	TabContainers
	TabHistory
)

type Model struct {
	state     SessionState
	mode      AppMode
	activeTab Tab
	width     int
	height    int
	quitting  bool
	tickCount int

	agent      agent.Model
	containers monitor.Model
	history    history.Model

	tabBar    components.TabBar
	statusBar components.StatusBar
	spinner   spinner.Model
	help      help.Model

	db       *sql.DB
	aiClient *llm.HybridClient
	pipe     *pipeline.Pipeline
	cwd      string
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	cwd, _ := os.Getwd()
	aiClient := llm.NewHybridClient()

	pipe := pipeline.NewPipeline()

	cmdPlugin := command.New()
	pipe.Register(cmdPlugin)

	aiPlug := ai.New(aiClient)
	pipe.Register(aiPlug)

	pipe.Start()

	pipe.State().SetCwd(cwd)

	tabBar := components.NewTabBar([]components.TabItem{
		{Icon: "◈", Label: "Agent"},
		{Icon: "⬢", Label: "Containers"},
		{Icon: "↻", Label: "History"},
	})

	return Model{
		state:     StateLoading,
		mode:      ModeNormal,
		activeTab: TabAgent,
		cwd:       cwd,
		aiClient:  aiClient,
		pipe:      pipe,

		agent:      agent.New(pipe),
		containers: monitor.New(),
		history:    history.New(),

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

		m.agent = m.agent.SetSize(msg.Width, msg.Height-4)
		m.containers = m.containers.SetSize(msg.Width, msg.Height-4)
		m.history = m.history.SetSize(msg.Width, msg.Height-4)

	case dockerHealthMsg:
		m.agent = m.agent.SetDockerHealth(msg.health)
		m.containers = m.containers.SetServices(msg.health.Containers)
		if msg.health.Available {
			m.state = StateMain
			if len(msg.health.Containers) > 0 {
				cmds = append(cmds, fetchContainerLogs(msg.health.Containers[0].ID))
			}
		}

	case containerLogsMsg:
		m.containers = m.containers.SetLogLines(msg.lines)

	case gpuStatsMsg:
		m.agent = m.agent.SetGPUStats(msg.stats)

	case serviceHealthMsg:
		_ = msg.services

	case historyLoadedMsg:
		if msg.err == nil {
			m.db = msg.db
			m.history = m.history.SetHistory(msg.history)
		}

	case starshipLineMsg:
		m.agent = m.agent.SetStarshipLine(msg.line)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

		m.tickCount++
		if m.tickCount >= 10 {
			m.tickCount = 0
			cmds = append(cmds, checkGPUStats, checkDockerHealth, checkServices, checkStarshipLine)
		}

	case agent.CommandExecutedMsg:
		var cmd tea.Cmd
		m.agent, cmd = m.agent.Update(msg, agent.DefaultKeyMap())
		m.mode = m.getModeFromTab()
		cmds = append(cmds, cmd)

	case agent.AIResponseMsg:
		var cmd tea.Cmd
		m.agent, cmd = m.agent.Update(msg, agent.DefaultKeyMap())
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if m.mode == ModeNormal {
			switch msg.String() {
			case "tab":
				m.activeTab = Tab((int(m.activeTab) + 1) % 3)
			case "shift+tab":
				m.activeTab = Tab((int(m.activeTab) + 2) % 3)
			case "1":
				m.activeTab = TabAgent
			case "2":
				m.activeTab = TabContainers
				if m.containers.SelectedService() != nil {
					cmds = append(cmds, fetchContainerLogs(m.containers.SelectedService().ID))
				}
			case "3":
				m.activeTab = TabHistory
			case "q":
				m.quitting = true
				return m, tea.Quit
			}
		}

		var cmd tea.Cmd
		switch m.activeTab {
		case TabAgent:
			m.agent, cmd = m.agent.Update(msg, agent.DefaultKeyMap())
			m.mode = m.getModeFromTab()
			cmds = append(cmds, cmd)

		case TabContainers:
			oldCursor := m.containers.ServicesList().Index()
			m.containers, cmd = m.containers.Update(msg, monitor.DefaultKeyMap())
			cmds = append(cmds, cmd)

			if m.containers.ServicesList().Index() != oldCursor {
				if svc := m.containers.SelectedService(); svc != nil {
					cmds = append(cmds, fetchContainerLogs(svc.ID))
				}
			}

		case TabHistory:
			m.history, cmd = m.history.Update(msg, history.DefaultKeyMap())
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) getModeFromTab() AppMode {
	switch m.activeTab {
	case TabAgent:
		if m.agent.InsertMode() {
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

	status := m.spinner.View() + " Initializing..."
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
	case TabAgent:
		content = m.agent.View()
	case TabContainers:
		content = m.containers.View()
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
	case TabAgent:
		statusBar = m.statusBar.Render(AgentKeys, focusLabel)
	case TabContainers:
		statusBar = m.statusBar.Render(MonitorKeys, focusLabel)
	case TabHistory:
		statusBar = m.statusBar.Render(HistoryKeys, focusLabel)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, styledContent, statusBar)
}

func (m Model) getFocusLabel() string {
	switch m.activeTab {
	case TabAgent:
		return "Agent"
	case TabContainers:
		switch m.containers.Focus() {
		case monitor.FocusServices:
			return "Services"
		case monitor.FocusImages:
			return "Images"
		case monitor.FocusLogs:
			return "Logs"
		case monitor.FocusStats:
			return "Stats"
		}
		return "Containers"
	case TabHistory:
		if m.history.Focus() == history.FocusSidebar {
			return "History"
		}
		return "Details"
	}
	return "Main"
}

type containerLogsMsg struct {
	containerID string
	lines       []string
	err         error
}

func fetchContainerLogs(containerID string) tea.Cmd {
	return func() tea.Msg {
		dockerClient, err := infra.GetSharedDockerClient()
		if err != nil {
			return containerLogsMsg{containerID: containerID, err: err}
		}

		lines, err := dockerClient.GetContainerLogs(context.Background(), containerID, 100)
		return containerLogsMsg{
			containerID: containerID,
			lines:       lines,
			err:         err,
		}
	}
}

func checkDockerHealth() tea.Msg {
	dockerClient, err := infra.GetSharedDockerClient()
	if err != nil {
		return dockerHealthMsg{
			health: infra.DockerHealth{
				Available: false,
				Error:     err,
			},
		}
	}

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

type starshipLineMsg struct {
	line string
}

func checkStarshipLine() tea.Msg {
	line := infra.GetStarshipStatusLine()
	return starshipLineMsg{line: line}
}

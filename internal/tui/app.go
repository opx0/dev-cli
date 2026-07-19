package tui

import (
	"context"
	"time"

	"dev-cli/internal/infra"
	"dev-cli/internal/storage"
	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/tabs/history"
	"dev-cli/internal/tui/tabs/monitor"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SessionState int

const (
	StateLoading SessionState = iota
	StateMain
)

type Tab int

const (
	TabContainers Tab = iota
	TabHistory
)

type refreshMsg struct{}

type Model struct {
	state     SessionState
	activeTab Tab
	width     int
	height    int
	quitting  bool

	containers monitor.Model
	history    history.Model
	tabBar     components.TabBar
	statusBar  components.StatusBar
	spinner    spinner.Model
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	return Model{
		state:      StateLoading,
		activeTab:  TabContainers,
		containers: monitor.New(),
		history:    history.New(),
		tabBar: components.NewTabBar([]components.TabItem{
			{Icon: "⬢", Label: "Containers"},
			{Icon: "↻", Label: "History"},
		}),
		statusBar: components.NewStatusBar(),
		spinner:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkDockerHealth, checkDBAndHistory, scheduleRefresh())
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return refreshMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.tabBar = m.tabBar.SetWidth(msg.Width)
		m.statusBar = m.statusBar.SetWidth(msg.Width)
		m.containers = m.containers.SetSize(msg.Width, msg.Height-4)
		m.history = m.history.SetSize(msg.Width, msg.Height-4)
	case dockerHealthMsg:
		m.containers = m.containers.SetDockerHealth(msg.health).SetServices(msg.health.Containers)
		m.state = StateMain
		if selected := m.containers.SelectedService(); selected != nil {
			cmds = append(cmds, fetchContainerLogs(selected.ID))
		}
	case containerLogsMsg:
		if selected := m.containers.SelectedService(); selected != nil && selected.ID == msg.containerID && msg.err == nil {
			m.containers = m.containers.SetLogLines(msg.lines)
		}
	case historyLoadedMsg:
		if msg.err == nil {
			m.history = m.history.SetHistory(msg.history)
		}
		m.state = StateMain
	case refreshMsg:
		cmds = append(cmds, checkDockerHealth, scheduleRefresh())
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}
		switch msg.String() {
		case "tab":
			m.activeTab = Tab((int(m.activeTab) + 1) % 2)
		case "shift+tab":
			m.activeTab = Tab((int(m.activeTab) + 1) % 2)
		case "1":
			m.activeTab = TabContainers
		case "2":
			m.activeTab = TabHistory
		}

		var cmd tea.Cmd
		if m.activeTab == TabContainers {
			oldCursor := m.containers.SelectedIndex()
			m.containers, cmd = m.containers.Update(msg, monitor.DefaultKeyMap())
			if m.containers.SelectedIndex() != oldCursor {
				if selected := m.containers.SelectedService(); selected != nil {
					cmds = append(cmds, fetchContainerLogs(selected.ID))
				}
			}
		} else {
			m.history, cmd = m.history.Update(msg, history.DefaultKeyMap())
		}
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if m.state == StateLoading {
		return "\n" + m.spinner.View() + " Initializing...\n"
	}

	m.tabBar = m.tabBar.SetActive(int(m.activeTab))
	content, focus := m.containers.View(), "Containers"
	status := m.statusBar.Render(MonitorKeys, focus)
	if m.activeTab == TabHistory {
		content, focus = m.history.View(), "History"
		status = m.statusBar.Render(HistoryKeys, focus)
	}
	height := m.height - 3
	if height < 10 {
		height = 10
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.tabBar.Render(),
		lipgloss.NewStyle().Height(height).MaxWidth(m.width).Render(content),
		status,
	)
}

func fetchContainerLogs(containerID string) tea.Cmd {
	return func() tea.Msg {
		client, err := infra.GetSharedDockerClient()
		if err != nil {
			return containerLogsMsg{containerID: containerID, err: err}
		}
		lines, err := client.GetContainerLogs(context.Background(), containerID, 100)
		return containerLogsMsg{containerID: containerID, lines: lines, err: err}
	}
}

func checkDockerHealth() tea.Msg {
	client, err := infra.GetSharedDockerClient()
	if err != nil {
		return dockerHealthMsg{health: infra.DockerHealth{Error: err}}
	}
	return dockerHealthMsg{health: client.CheckHealth(context.Background())}
}

func checkDBAndHistory() tea.Msg {
	db, err := storage.InitDB()
	if err != nil {
		return historyLoadedMsg{err: err}
	}
	defer func() { _ = db.Close() }()
	items, err := storage.GetRecentHistory(db, 50)
	return historyLoadedMsg{history: items, err: err}
}

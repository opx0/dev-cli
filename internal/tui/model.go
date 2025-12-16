package tui

import (
	"context"
	"os"

	"database/sql"
	"dev-cli/internal/config"
	"dev-cli/internal/infra"
	"dev-cli/internal/llm"
	"dev-cli/internal/storage"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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

type FocusPanel int

const (
	FocusSidebar FocusPanel = iota
	FocusMain
)

type Tab int

const (
	TabDashboard Tab = iota
	TabMonitor
	TabAssist
	TabHistory
)

type Model struct {
	state         SessionState
	mode          AppMode
	activeTab     Tab
	focus         FocusPanel
	spinner       spinner.Model
	input         textinput.Model
	viewport      viewport.Model // Generic viewport for logs/chat/history
	width         int
	height        int
	cwd           string
	dockerHealth  infra.DockerHealth
	gpuStats      infra.GPUStats
	serviceHealth []infra.ServiceStatus
	quitting      bool

	// Monitor Tab State
	monitorCursor int

	// Assist Tab State
	chatHistory []string // Placeholder for chat messages
	aiClient    *llm.HybridClient
	aiMode      string // "local" or "cloud"

	// History Tab State
	db             *sql.DB
	commandHistory []storage.HistoryItem
	historyCursor  int
}

type dockerHealthMsg struct {
	health infra.DockerHealth
}

type gpuStatsMsg struct {
	stats infra.GPUStats
}

type serviceHealthMsg struct {
	services []infra.ServiceStatus
}

type commandOutputMsg string

type errMsg error

type clearViewportMsg struct{}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	ti := textinput.New()
	ti.Placeholder = "type command... (press 'i' to insert)"
	ti.CharLimit = 256
	ti.Width = 60

	vp := viewport.New(0, 0)

	cwd, _ := os.Getwd()

	// Initialize AI Client
	aiClient := llm.NewHybridClient()

	aiMode := "local"
	if config.Current.IsWebSearchEnabled() {
		aiMode = "cloud"
	}

	return Model{
		state:     StateLoading,
		mode:      ModeNormal,
		activeTab: TabDashboard,
		focus:     FocusSidebar,
		spinner:   s,
		input:     ti,
		viewport:  vp,
		cwd:       cwd,
		aiClient:  aiClient,
		aiMode:    aiMode,
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
		// If fails to get history, we still return DB so app can use it?
		// Or maybe valid DB but error reading.
		return historyLoadedMsg{db: db, err: err}
	}

	return historyLoadedMsg{db: db, history: history}
}

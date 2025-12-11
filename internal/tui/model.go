package tui

import (
	"context"
	"os"

	"dev-cli/internal/infra"

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

// New: Define Input Modes (Vim concept)
type AppMode int

const (
	ModeNormal AppMode = iota // Navigation (j, k, 1, 2)
	ModeInsert                // Typing (full keyboard input)
)

type FocusPanel int

const (
	FocusContainers FocusPanel = iota
	FocusObservability
	FocusKeybinds
	FocusTerminal
)

type Model struct {
	state        SessionState
	mode         AppMode // Track current mode
	focus        FocusPanel
	spinner      spinner.Model
	input        textinput.Model
	viewport     viewport.Model // Use Viewport for terminal history
	width        int
	height       int
	cwd          string // Current working directory
	dockerHealth infra.DockerHealth
	quitting     bool
	err          error
}

type dockerHealthMsg struct {
	health infra.DockerHealth
}

// Msg to display command output in viewport
type commandOutputMsg string

// Msg to handle errors from Exec
type errMsg error

// Msg to clear the viewport
type clearViewportMsg struct{}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	ti := textinput.New()
	ti.Placeholder = "type command... (press 'i' to insert)"
	ti.CharLimit = 256
	ti.Width = 60

	vp := viewport.New(0, 0) // Dimensions set in Update on resize

	cwd, _ := os.Getwd()

	return Model{
		state:    StateLoading,
		mode:     ModeNormal, // Start in Normal mode
		focus:    FocusContainers,
		spinner:  s,
		input:    ti,
		viewport: vp,
		cwd:      cwd,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerHealth,
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

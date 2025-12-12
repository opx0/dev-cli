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

type AppMode int

const (
	ModeNormal AppMode = iota
	ModeInsert
)

type FocusPanel int

const (
	FocusStatus FocusPanel = iota
	FocusKeybinds
	FocusTerminal
)

type Model struct {
	state        SessionState
	mode         AppMode
	focus        FocusPanel
	spinner      spinner.Model
	input        textinput.Model
	viewport     viewport.Model
	width        int
	height       int
	cwd          string
	dockerHealth infra.DockerHealth
	quitting     bool
}

type dockerHealthMsg struct {
	health infra.DockerHealth
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

	return Model{
		state:    StateLoading,
		mode:     ModeNormal,
		focus:    FocusStatus,
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

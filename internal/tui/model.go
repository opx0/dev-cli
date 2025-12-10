package tui

import (
	"context"

	"dev-cli/internal/infra"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SessionState int

const (
	StateLoading SessionState = iota
	StateMain
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
	focus        FocusPanel
	spinner      spinner.Model
	input        textinput.Model
	width        int
	height       int
	dockerHealth infra.DockerHealth
	cmdHistory   []string
	quitting     bool
}

type dockerHealthMsg struct {
	health infra.DockerHealth
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	ti := textinput.New()
	ti.Placeholder = "type command..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	return Model{
		state:      StateLoading,
		focus:      FocusTerminal,
		spinner:    s,
		input:      ti,
		cmdHistory: []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textinput.Blink,
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

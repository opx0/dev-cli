package dashboard

import (
	"dev-cli/internal/infra"
	"dev-cli/internal/tui/components"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type FocusPanel int

const (
	FocusSidebar FocusPanel = iota
	FocusMain
)

type Model struct {
	width  int
	height int
	focus  FocusPanel

	sidebar  components.Panel
	terminal components.Panel
	viewport viewport.Model
	input    textinput.Model

	dockerHealth  infra.DockerHealth
	gpuStats      infra.GPUStats
	serviceHealth []infra.ServiceStatus
	cwd           string

	insertMode bool
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "type command... (press 'i' to insert)"
	ti.CharLimit = 256
	ti.Width = 60

	vp := viewport.New(0, 0)

	return Model{
		sidebar:  components.NewPanel(" ⌘ Mission Control"),
		terminal: components.NewPanel(" >_ Terminal"),
		viewport: vp,
		input:    ti,
		focus:    FocusSidebar,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	sidebarWidth := 30
	terminalWidth := w - sidebarWidth - 4
	panelHeight := h - 6

	if terminalWidth < 40 {
		terminalWidth = 40
	}
	if panelHeight < 10 {
		panelHeight = 10
	}

	m.sidebar = m.sidebar.SetSize(sidebarWidth, panelHeight)
	m.terminal = m.terminal.SetSize(terminalWidth, panelHeight)
	m.viewport.Width = terminalWidth - 4
	m.viewport.Height = panelHeight - 6
	m.input.Width = terminalWidth - 10

	return m
}

func (m Model) SetFocus(f FocusPanel) Model {
	m.focus = f
	return m
}

func (m Model) SetInsertMode(insert bool) Model {
	m.insertMode = insert
	if insert {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
	return m
}

func (m Model) SetDockerHealth(h infra.DockerHealth) Model {
	m.dockerHealth = h
	return m
}

func (m Model) SetGPUStats(s infra.GPUStats) Model {
	m.gpuStats = s
	return m
}

func (m Model) SetServiceHealth(s []infra.ServiceStatus) Model {
	m.serviceHealth = s
	return m
}

func (m Model) SetCwd(cwd string) Model {
	m.cwd = cwd
	return m
}

func (m Model) Focus() FocusPanel {
	return m.focus
}

func (m Model) InsertMode() bool {
	return m.insertMode
}

func (m Model) Input() textinput.Model {
	return m.input
}

func (m Model) SetInput(ti textinput.Model) Model {
	m.input = ti
	return m
}

func (m Model) Viewport() viewport.Model {
	return m.viewport
}

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) ClearViewport() Model {
	m.viewport.SetContent("")
	return m
}

func (m Model) AppendOutput(text string) Model {
	m.viewport.SetContent(m.viewport.View() + text)
	m.viewport.GotoBottom()
	return m
}

func (m Model) Width() int {
	return m.width
}

func (m Model) Height() int {
	return m.height
}

func (m Model) DockerHealth() infra.DockerHealth {
	return m.dockerHealth
}

func (m Model) GPUStats() infra.GPUStats {
	return m.gpuStats
}

func (m Model) ServiceHealth() []infra.ServiceStatus {
	return m.serviceHealth
}

func (m Model) Cwd() string {
	return m.cwd
}

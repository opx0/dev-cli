package dashboard

import (
	"dev-cli/internal/infra"
	"dev-cli/internal/tui/components"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type OutputBlock struct {
	Command   string
	Output    string
	ExitCode  int
	Timestamp string
	Folded    bool
}

type Model struct {
	width  int
	height int

	viewport viewport.Model
	input    textinput.Model

	dockerHealth  infra.DockerHealth
	gpuStats      infra.GPUStats
	serviceHealth []infra.ServiceStatus
	cwd           string

	outputBlocks  []OutputBlock
	selectedBlock int

	insertMode     bool
	showingActions bool
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "type command..."
	ti.CharLimit = 512
	ti.Width = 60

	vp := viewport.New(0, 0)

	return Model{
		viewport:      vp,
		input:         ti,
		outputBlocks:  []OutputBlock{},
		selectedBlock: -1,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	contentHeight := h - 10 // header + input area
	if contentHeight < 5 {
		contentHeight = 5
	}

	m.viewport.Width = w - 4
	m.viewport.Height = contentHeight
	m.input.Width = w - 10

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

func (m Model) SetInsertMode(insert bool) Model {
	m.insertMode = insert
	if insert {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
	return m
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

func (m Model) AddOutputBlock(cmd string) Model {
	block := OutputBlock{
		Command: cmd,
	}
	m.outputBlocks = append(m.outputBlocks, block)
	m.selectedBlock = len(m.outputBlocks) - 1
	return m
}

func (m Model) UpdateLastBlock(output string, exitCode int) Model {
	if len(m.outputBlocks) > 0 {
		idx := len(m.outputBlocks) - 1
		m.outputBlocks[idx].Output = output
		m.outputBlocks[idx].ExitCode = exitCode
	}
	return m
}

func (m Model) OutputBlocks() []OutputBlock {
	return m.outputBlocks
}

func (m Model) SelectedBlock() int {
	return m.selectedBlock
}

func (m Model) SetSelectedBlock(idx int) Model {
	if idx >= -1 && idx < len(m.outputBlocks) {
		m.selectedBlock = idx
	}
	return m
}

func (m Model) ToggleFoldBlock(idx int) Model {
	if idx >= 0 && idx < len(m.outputBlocks) {
		m.outputBlocks[idx].Folded = !m.outputBlocks[idx].Folded
	}
	return m
}

func (m Model) ShowingActions() bool {
	return m.showingActions
}

func (m Model) SetShowingActions(show bool) Model {
	m.showingActions = show
	return m
}

func (m Model) HeaderWidgets() []components.HeaderWidget {
	var widgets []components.HeaderWidget

	dockerWidget := components.NewHeaderWidget("🐳", "Docker", "")
	if m.dockerHealth.Available {
		running := 0
		for _, c := range m.dockerHealth.Containers {
			if c.State == "running" {
				running++
			}
		}
		dockerWidget.Value = string(rune('0'+running)) + " ●"
		dockerWidget = dockerWidget.SetSuccess(true)
	} else {
		dockerWidget.Value = "off"
		dockerWidget = dockerWidget.SetError(true)
	}
	widgets = append(widgets, dockerWidget)

	if m.gpuStats.Available {
		gpuWidget := components.NewHeaderWidget("▮", "GPU", "")
		gpuWidget.Value = string(rune('0'+m.gpuStats.UtilizationPct/10)) + "0%"
		if m.gpuStats.UtilizationPct > 80 {
			gpuWidget = gpuWidget.SetError(true)
		} else {
			gpuWidget = gpuWidget.SetActive(true)
		}
		widgets = append(widgets, gpuWidget)
	}

	onlineCount := 0
	for _, s := range m.serviceHealth {
		if s.Available {
			onlineCount++
		}
	}
	if len(m.serviceHealth) > 0 {
		svcWidget := components.NewHeaderWidget("◉", "Services", "")
		svcWidget.Value = string(rune('0'+onlineCount)) + "/" + string(rune('0'+len(m.serviceHealth)))
		if onlineCount == len(m.serviceHealth) {
			svcWidget = svcWidget.SetSuccess(true)
		} else {
			svcWidget = svcWidget.SetError(true)
		}
		widgets = append(widgets, svcWidget)
	}

	return widgets
}

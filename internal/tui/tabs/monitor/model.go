package monitor

import (
	"dev-cli/internal/infra"
	"dev-cli/internal/tui/components"

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
	logPanel components.Panel
	viewport viewport.Model

	dockerHealth infra.DockerHealth
	logLines     []string

	containerCursor  int
	horizontalOffset int
	wrapMode         bool
	maxLineWidth     int
}

func New() Model {
	vp := viewport.New(0, 0)

	return Model{
		sidebar:  components.NewPanel(" ▢ Containers"),
		logPanel: components.NewPanel(" ☰ Logs"),
		viewport: vp,
		focus:    FocusSidebar,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	sidebarWidth := 30
	logWidth := w - sidebarWidth - 4
	panelHeight := h - 6

	if logWidth < 40 {
		logWidth = 40
	}
	if panelHeight < 10 {
		panelHeight = 10
	}

	m.sidebar = m.sidebar.SetSize(sidebarWidth, panelHeight)
	m.logPanel = m.logPanel.SetSize(logWidth, panelHeight)
	m.viewport.Width = logWidth - 4
	m.viewport.Height = panelHeight - 4

	return m
}

func (m Model) SetFocus(f FocusPanel) Model {
	m.focus = f
	return m
}

func (m Model) SetDockerHealth(h infra.DockerHealth) Model {
	m.dockerHealth = h
	return m
}

func (m Model) SetLogLines(lines []string) Model {
	m.logLines = lines

	m.maxLineWidth = 0
	for _, line := range lines {
		if len(line) > m.maxLineWidth {
			m.maxLineWidth = len(line)
		}
	}
	return m
}

func (m Model) Focus() FocusPanel {
	return m.focus
}

func (m Model) ContainerCursor() int {
	return m.containerCursor
}

func (m Model) SetContainerCursor(c int) Model {
	m.containerCursor = c
	return m
}

func (m Model) HorizontalOffset() int {
	return m.horizontalOffset
}

func (m Model) SetHorizontalOffset(o int) Model {
	m.horizontalOffset = o
	return m
}

func (m Model) WrapMode() bool {
	return m.wrapMode
}

func (m Model) SetWrapMode(w bool) Model {
	m.wrapMode = w
	if w {
		m.horizontalOffset = 0
	}
	return m
}

func (m Model) ToggleWrapMode() Model {
	return m.SetWrapMode(!m.wrapMode)
}

func (m Model) DockerHealth() infra.DockerHealth {
	return m.dockerHealth
}

func (m Model) LogLines() []string {
	return m.logLines
}

func (m Model) MaxLineWidth() int {
	return m.maxLineWidth
}

func (m Model) Width() int {
	return m.width
}

func (m Model) Height() int {
	return m.height
}

func (m Model) Viewport() viewport.Model {
	return m.viewport
}

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) ContainerCount() int {
	return len(m.dockerHealth.Containers)
}

func (m Model) SelectedContainer() *infra.ContainerInfo {
	if m.containerCursor >= 0 && m.containerCursor < len(m.dockerHealth.Containers) {
		return &m.dockerHealth.Containers[m.containerCursor]
	}
	return nil
}

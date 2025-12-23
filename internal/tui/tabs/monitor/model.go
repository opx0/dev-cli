package monitor

import (
	"dev-cli/internal/infra"

	"github.com/charmbracelet/bubbles/viewport"
)

type FocusPanel int

const (
	FocusList FocusPanel = iota
	FocusLogs
	FocusStats
)

type SubPanel int

const (
	SubPanelContainers SubPanel = iota
	SubPanelImages
	SubPanelVolumes
)

type ContainerStats struct {
	CPUHistory []int
	MemUsed    int
	MemTotal   int
	NetIn      int64
	NetOut     int64
}

type Model struct {
	width  int
	height int
	focus  FocusPanel

	viewport viewport.Model

	dockerHealth   infra.DockerHealth
	logLines       []string
	containerStats map[string]ContainerStats

	containerCursor  int
	horizontalOffset int
	wrapMode         bool
	maxLineWidth     int

	followMode      bool
	logLevelFilter  string // "", "ERROR", "WARN", "INFO"
	showingActions  bool
	actionMenuIndex int

	// New fields for lazydocker-style features
	subPanel      SubPanel
	images        []infra.ImageInfo
	volumes       []infra.VolumeInfo
	imageCursor   int
	volumeCursor  int
	pendingAction string // action waiting for confirmation (e.g., "remove")
	statusMessage string // temporary status message
}

func New() Model {
	vp := viewport.New(0, 0)

	return Model{
		viewport:       vp,
		focus:          FocusList,
		containerStats: make(map[string]ContainerStats),
		subPanel:       SubPanelContainers,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	listWidth := 28
	if w < 100 {
		listWidth = 24
	}

	logWidth := w - listWidth - 4
	if logWidth < 40 {
		logWidth = 40
	}

	panelHeight := h - 4
	statsHeight := 6
	logHeight := panelHeight - statsHeight - 2

	if logHeight < 10 {
		logHeight = 10
	}

	m.viewport.Width = logWidth - 4
	m.viewport.Height = logHeight - 4

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

func (m Model) FollowMode() bool {
	return m.followMode
}

func (m Model) SetFollowMode(follow bool) Model {
	m.followMode = follow
	if follow {
		m.viewport.GotoBottom()
	}
	return m
}

func (m Model) ToggleFollowMode() Model {
	return m.SetFollowMode(!m.followMode)
}

func (m Model) LogLevelFilter() string {
	return m.logLevelFilter
}

func (m Model) SetLogLevelFilter(level string) Model {
	m.logLevelFilter = level
	return m
}

func (m Model) CycleLogLevelFilter() Model {
	switch m.logLevelFilter {
	case "":
		m.logLevelFilter = "ERROR"
	case "ERROR":
		m.logLevelFilter = "WARN"
	case "WARN":
		m.logLevelFilter = "INFO"
	case "INFO":
		m.logLevelFilter = ""
	}
	return m
}

func (m Model) ShowingActions() bool {
	return m.showingActions
}

func (m Model) SetShowingActions(show bool) Model {
	m.showingActions = show
	if !show {
		m.actionMenuIndex = 0
	}
	return m
}

func (m Model) ActionMenuIndex() int {
	return m.actionMenuIndex
}

func (m Model) SetActionMenuIndex(idx int) Model {
	m.actionMenuIndex = idx
	return m
}

func (m Model) ContainerStats() map[string]ContainerStats {
	return m.containerStats
}

func (m Model) SetContainerStats(name string, stats ContainerStats) Model {
	if m.containerStats == nil {
		m.containerStats = make(map[string]ContainerStats)
	}
	m.containerStats[name] = stats
	return m
}

func (m Model) GetSelectedContainerStats() ContainerStats {
	if container := m.SelectedContainer(); container != nil {
		if stats, ok := m.containerStats[container.Name]; ok {
			return stats
		}
	}
	return ContainerStats{}
}

// SubPanel methods
func (m Model) SubPanel() SubPanel {
	return m.subPanel
}

func (m Model) SetSubPanel(sp SubPanel) Model {
	m.subPanel = sp
	return m
}

// Images methods
func (m Model) Images() []infra.ImageInfo {
	return m.images
}

func (m Model) SetImages(images []infra.ImageInfo) Model {
	m.images = images
	return m
}

func (m Model) ImageCursor() int {
	return m.imageCursor
}

func (m Model) SetImageCursor(c int) Model {
	m.imageCursor = c
	return m
}

// Volumes methods
func (m Model) Volumes() []infra.VolumeInfo {
	return m.volumes
}

func (m Model) SetVolumes(volumes []infra.VolumeInfo) Model {
	m.volumes = volumes
	return m
}

func (m Model) VolumeCursor() int {
	return m.volumeCursor
}

func (m Model) SetVolumeCursor(c int) Model {
	m.volumeCursor = c
	return m
}

// Pending action methods
func (m Model) PendingAction() string {
	return m.pendingAction
}

func (m Model) SetPendingAction(action string) Model {
	m.pendingAction = action
	return m
}

// Status message methods
func (m Model) StatusMessage() string {
	return m.statusMessage
}

func (m Model) SetStatusMessage(msg string) Model {
	m.statusMessage = msg
	return m
}

func (m Model) ClearStatusMessage() Model {
	m.statusMessage = ""
	return m
}

package monitor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"dev-cli/internal/infra"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FocusPanel int

const (
	FocusServices FocusPanel = iota
	FocusImages
	FocusLogs
	FocusStats
)

// Service item for bubbles/list
type serviceItem struct {
	info infra.ContainerInfo
}

func (i serviceItem) Title() string       { return i.info.Name }
func (i serviceItem) Description() string { return i.info.Status }
func (i serviceItem) FilterValue() string { return i.info.Name }

// Image item for bubbles/list
type imageItem struct {
	info infra.ImageInfo
}

func (i imageItem) Title() string {
	if len(i.info.Tags) > 0 {
		return i.info.Tags[0]
	}
	return i.info.ID
}
func (i imageItem) Description() string { return formatSize(i.info.Size) }
func (i imageItem) FilterValue() string { return i.Title() }

// Custom delegate for service list
type serviceDelegate struct{}

func (d serviceDelegate) Height() int                             { return 1 }
func (d serviceDelegate) Spacing() int                            { return 0 }
func (d serviceDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d serviceDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(serviceItem)
	if !ok {
		return
	}

	status := "●"
	statusColor := theme.Green
	if i.info.State != "running" {
		status = "○"
		statusColor = theme.Red
	}

	name := i.info.Name
	maxWidth := m.Width() - 6
	if maxWidth < 5 {
		maxWidth = 5
	}
	if len(name) > maxWidth {
		name = name[:maxWidth-1] + "…"
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	textStyle := lipgloss.NewStyle().Foreground(theme.Text)

	line := fmt.Sprintf(" %s %s", statusStyle.Render(status), textStyle.Render(name))

	if index == m.Index() {
		line = lipgloss.NewStyle().
			Background(theme.Surface1).
			Foreground(theme.Lavender).
			Bold(true).
			Width(m.Width()).
			Render(line)
	}

	fmt.Fprint(w, line)
}

// Custom delegate for image list
type imageDelegate struct{}

func (d imageDelegate) Height() int                             { return 1 }
func (d imageDelegate) Spacing() int                            { return 0 }
func (d imageDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d imageDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(imageItem)
	if !ok {
		return
	}

	tag := i.Title()
	maxWidth := m.Width() - 4
	if maxWidth < 5 {
		maxWidth = 5
	}
	if len(tag) > maxWidth {
		tag = tag[:maxWidth-1] + "…"
	}

	textStyle := lipgloss.NewStyle().Foreground(theme.Text)
	line := fmt.Sprintf(" %s", textStyle.Render(tag))

	if index == m.Index() {
		line = lipgloss.NewStyle().
			Background(theme.Surface1).
			Foreground(theme.Lavender).
			Bold(true).
			Width(m.Width()).
			Render(line)
	}

	fmt.Fprint(w, line)
}

// Stats for display
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

	// Lists (using bubbles/list like history sidebar)
	servicesList list.Model
	imagesList   list.Model
	viewport     viewport.Model

	// Data
	services       []infra.ContainerInfo
	images         []infra.ImageInfo
	logLines       []string
	containerStats map[string]ContainerStats

	// Log recording
	isRecording   bool
	recordingFile *os.File
	recordingPath string

	// UI state
	followMode     bool
	logLevelFilter string
}

func New() Model {

	sDelegate := serviceDelegate{}
	sList := list.New([]list.Item{}, sDelegate, 0, 0)
	sList.SetShowHelp(false)
	sList.SetShowTitle(false)
	sList.SetShowStatusBar(false)
	sList.SetFilteringEnabled(false)
	sList.DisableQuitKeybindings()
	sList.Styles.NoItems = lipgloss.NewStyle().Foreground(theme.Overlay0).Padding(1)

	iDelegate := imageDelegate{}
	iList := list.New([]list.Item{}, iDelegate, 0, 0)
	iList.SetShowHelp(false)
	iList.SetShowTitle(false)
	iList.SetShowStatusBar(false)
	iList.SetFilteringEnabled(false)
	iList.DisableQuitKeybindings()
	iList.Styles.NoItems = lipgloss.NewStyle().Foreground(theme.Overlay0).Padding(1)

	vp := viewport.New(0, 0)

	return Model{
		servicesList:   sList,
		imagesList:     iList,
		viewport:       vp,
		focus:          FocusServices,
		containerStats: make(map[string]ContainerStats),
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	sidebarWidth := 28
	if w < 100 {
		sidebarWidth = 24
	}

	panelHeight := h - 4
	servicesHeight := (panelHeight - 8) / 2
	imagesHeight := (panelHeight - 8) / 2
	_ = 6

	if servicesHeight < 5 {
		servicesHeight = 5
	}
	if imagesHeight < 5 {
		imagesHeight = 5
	}

	m.servicesList.SetWidth(sidebarWidth - 4)
	m.servicesList.SetHeight(servicesHeight - 2)
	m.imagesList.SetWidth(sidebarWidth - 4)
	m.imagesList.SetHeight(imagesHeight - 2)

	logWidth := w - sidebarWidth - 4
	if logWidth < 40 {
		logWidth = 40
	}
	m.viewport.Width = logWidth - 4
	m.viewport.Height = panelHeight - 4

	return m
}

// SetServices updates the services list
func (m Model) SetServices(containers []infra.ContainerInfo) Model {
	m.services = containers

	items := make([]list.Item, len(containers))
	for i, c := range containers {
		items[i] = serviceItem{info: c}
	}
	m.servicesList.SetItems(items)

	return m
}

// SetImages updates the images list
func (m Model) SetImages(images []infra.ImageInfo) Model {
	m.images = images

	items := make([]list.Item, len(images))
	for i, img := range images {
		items[i] = imageItem{info: img}
	}
	m.imagesList.SetItems(items)

	return m
}

// SetLogLines updates the log content
func (m Model) SetLogLines(lines []string) Model {
	m.logLines = lines

	if m.isRecording && m.recordingFile != nil {
		for _, line := range lines {
			m.recordingFile.WriteString(line + "\n")
		}
	}

	return m
}

// Recording methods
func (m Model) StartRecording() Model {
	if m.isRecording {
		return m
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return m
	}

	logDir := filepath.Join(homeDir, ".devlogs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return m
	}

	serviceName := "unknown"
	if sel := m.servicesList.SelectedItem(); sel != nil {
		if s, ok := sel.(serviceItem); ok {
			serviceName = s.info.Name
		}
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("docker-%s-%s.log", serviceName, timestamp)
	m.recordingPath = filepath.Join(logDir, filename)

	file, err := os.Create(m.recordingPath)
	if err != nil {
		return m
	}

	m.recordingFile = file
	m.isRecording = true

	m.recordingFile.WriteString(fmt.Sprintf("# Docker Log Recording: %s\n", serviceName))
	m.recordingFile.WriteString(fmt.Sprintf("# Started: %s\n\n", time.Now().Format(time.RFC3339)))

	return m
}

func (m Model) StopRecording() Model {
	if !m.isRecording {
		return m
	}

	if m.recordingFile != nil {
		m.recordingFile.WriteString(fmt.Sprintf("\n# Stopped: %s\n", time.Now().Format(time.RFC3339)))
		m.recordingFile.Close()
		m.recordingFile = nil
	}

	m.isRecording = false
	return m
}

func (m Model) ToggleRecording() Model {
	if m.isRecording {
		return m.StopRecording()
	}
	return m.StartRecording()
}

func (m Model) IsRecording() bool {
	return m.isRecording
}

func (m Model) RecordingPath() string {
	return m.recordingPath
}

// Getters
func (m Model) Focus() FocusPanel               { return m.focus }
func (m Model) SetFocus(f FocusPanel) Model     { m.focus = f; return m }
func (m Model) Width() int                      { return m.width }
func (m Model) Height() int                     { return m.height }
func (m Model) Services() []infra.ContainerInfo { return m.services }
func (m Model) Images() []infra.ImageInfo       { return m.images }
func (m Model) LogLines() []string              { return m.logLines }
func (m Model) Viewport() viewport.Model        { return m.viewport }
func (m Model) ServicesList() list.Model        { return m.servicesList }
func (m Model) ImagesList() list.Model          { return m.imagesList }
func (m Model) FollowMode() bool                { return m.followMode }
func (m Model) LogLevelFilter() string          { return m.logLevelFilter }

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) SetServicesList(l list.Model) Model {
	m.servicesList = l
	return m
}

func (m Model) SetImagesList(l list.Model) Model {
	m.imagesList = l
	return m
}

func (m Model) SetFollowMode(f bool) Model {
	m.followMode = f
	if f {
		m.viewport.GotoBottom()
	}
	return m
}

func (m Model) ToggleFollowMode() Model {
	return m.SetFollowMode(!m.followMode)
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

// Selected items
func (m Model) SelectedService() *infra.ContainerInfo {
	if sel := m.servicesList.SelectedItem(); sel != nil {
		if s, ok := sel.(serviceItem); ok {
			return &s.info
		}
	}
	return nil
}

func (m Model) SelectedImage() *infra.ImageInfo {
	if sel := m.imagesList.SelectedItem(); sel != nil {
		if i, ok := sel.(imageItem); ok {
			return &i.info
		}
	}
	return nil
}

// Stats
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

func (m Model) GetSelectedServiceStats() ContainerStats {
	if svc := m.SelectedService(); svc != nil {
		if stats, ok := m.containerStats[svc.Name]; ok {
			return stats
		}
	}
	return ContainerStats{}
}

// Helper
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

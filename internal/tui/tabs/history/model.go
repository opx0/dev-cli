package history

import (
	"fmt"
	"io"
	"strings"
	"time"

	"dev-cli/internal/storage"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FocusPanel int

const (
	FocusSidebar FocusPanel = iota
	FocusMain
)

type historyItem struct {
	storage.HistoryItem
}

func (i historyItem) Title() string       { return i.Command }
func (i historyItem) Description() string { return i.Timestamp.Format("15:04:05") }
func (i historyItem) FilterValue() string { return i.Command }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(historyItem)
	if !ok {
		return
	}

	icon := "✓"
	iconColor := theme.Green
	if i.ExitCode != 0 {
		icon = "✕"
		iconColor = theme.Red
	}

	cmd := i.Command
	maxWidth := m.Width() - 8
	if maxWidth < 10 {
		maxWidth = 10
	}
	if len(cmd) > maxWidth {
		cmd = cmd[:maxWidth-1] + "…"
	}

	iconStyle := lipgloss.NewStyle().Foreground(iconColor)
	textStyle := lipgloss.NewStyle().Foreground(theme.Text)
	line := fmt.Sprintf(" %s %s", iconStyle.Render(icon), textStyle.Render(cmd))

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

type Model struct {
	width    int
	height   int
	focus    FocusPanel
	list     list.Model
	viewport viewport.Model
	history  []storage.HistoryItem
}

func New() Model {
	delegate := itemDelegate{}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(theme.Overlay0).Padding(1)

	vp := viewport.New(0, 0)

	return Model{
		list:     l,
		viewport: vp,
		focus:    FocusSidebar,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	sidebarWidth := 40
	if w < 100 {
		sidebarWidth = w / 3
	}
	if sidebarWidth < 25 {
		sidebarWidth = 25
	}

	detailsWidth := w - sidebarWidth - 6
	panelHeight := h - 4

	if detailsWidth < 30 {
		detailsWidth = 30
	}
	if panelHeight < 10 {
		panelHeight = 10
	}

	m.list.SetWidth(sidebarWidth - 2)
	m.list.SetHeight(panelHeight - 4)
	m.viewport.Width = detailsWidth - 4
	m.viewport.Height = panelHeight - 4

	m.updateDetailsContent()
	return m
}

func (m Model) SetFocus(f FocusPanel) Model {
	m.focus = f
	return m
}

func (m Model) SetHistory(items []storage.HistoryItem) Model {
	m.history = items

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = historyItem{item}
	}
	m.list.SetItems(listItems)
	m.updateDetailsContent()
	return m
}

func (m *Model) updateDetailsContent() {
	if sel := m.list.SelectedItem(); sel != nil {
		if item, ok := sel.(historyItem); ok {
			content := m.formatDetails(item.HistoryItem)
			m.viewport.SetContent(content)
		}
	} else if len(m.history) > 0 {
		content := m.formatDetails(m.history[0])
		m.viewport.SetContent(content)
	} else {
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Padding(2).
			Render("No history items"))
	}
}

func (m Model) formatDetails(item storage.HistoryItem) string {
	labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Bold(true).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	codeStyle := lipgloss.NewStyle().Foreground(theme.Lavender).Background(theme.Surface0).Padding(0, 1)
	wrapStyle := lipgloss.NewStyle().Foreground(theme.Text).Width(m.viewport.Width - 2)

	exitStyle := valueStyle.Copy()
	if item.ExitCode != 0 {
		exitStyle = exitStyle.Foreground(theme.Red).Bold(true)
	} else {
		exitStyle = exitStyle.Foreground(theme.Green)
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render("Time"))
	b.WriteString(valueStyle.Render(item.Timestamp.Format(time.RFC822)) + "\n")
	b.WriteString(labelStyle.Render("Duration"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%dms", item.DurationMs)) + "\n")
	b.WriteString(labelStyle.Render("Exit Code"))
	b.WriteString(exitStyle.Render(fmt.Sprintf("%d", item.ExitCode)) + "\n\n")
	b.WriteString(labelStyle.Render("Command") + "\n")
	b.WriteString(codeStyle.Render(item.Command) + "\n")
	if item.Details != "" {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Output") + "\n")
		b.WriteString(wrapStyle.Render(item.Details))
	}
	return b.String()
}

func (m Model) Focus() FocusPanel { return m.focus }

func (m Model) Cursor() int { return m.list.Index() }

func (m Model) History() []storage.HistoryItem { return m.history }

func (m Model) HistoryCount() int { return len(m.history) }

func (m Model) SelectedItem() *storage.HistoryItem {
	if sel := m.list.SelectedItem(); sel != nil {
		if item, ok := sel.(historyItem); ok {
			return &item.HistoryItem
		}
	}
	return nil
}

func (m Model) Viewport() viewport.Model { return m.viewport }

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) List() list.Model { return m.list }

func (m Model) SetList(l list.Model) Model {
	m.list = l
	return m
}

func (m Model) Width() int { return m.width }

func (m Model) Height() int { return m.height }

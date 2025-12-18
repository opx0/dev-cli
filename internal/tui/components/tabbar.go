package components

import (
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

type TabBar struct {
	Tabs       []string
	ActiveTab  int
	Width      int
	ShowMode   bool
	InsertMode bool
}

func NewTabBar(tabs []string) TabBar {
	return TabBar{
		Tabs:      tabs,
		ActiveTab: 0,
	}
}

func (t TabBar) SetActive(idx int) TabBar {
	if idx >= 0 && idx < len(t.Tabs) {
		t.ActiveTab = idx
	}
	return t
}

func (t TabBar) SetWidth(w int) TabBar {
	t.Width = w
	return t
}

func (t TabBar) SetInsertMode(insert bool) TabBar {
	t.InsertMode = insert
	t.ShowMode = insert
	return t
}

func (t TabBar) Render() string {
	var renderedTabs []string

	for i, tab := range t.Tabs {
		style := theme.Tab
		if i == t.ActiveTab {
			style = theme.ActiveTab
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)

	modeStr := ""
	if t.InsertMode {
		modeStr = theme.ModeIndicator.Render(" INSERT ")
	}

	spacer := ""
	spacerWidth := t.Width - lipgloss.Width(row) - lipgloss.Width(modeStr) - 2
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	return row + spacer + modeStr
}

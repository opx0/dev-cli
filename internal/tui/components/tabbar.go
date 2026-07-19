package components

import (
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

type TabBar struct {
	Tabs      []TabItem
	ActiveTab int
	Width     int
}

type TabItem struct {
	Icon  string
	Label string
}

func NewTabBar(tabs []TabItem) TabBar {
	return TabBar{Tabs: tabs}
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

func (t TabBar) Render() string {
	var renderedTabs []string

	for i, tab := range t.Tabs {
		var style lipgloss.Style
		if i == t.ActiveTab {
			style = theme.ActiveTab
		} else {
			style = theme.Tab
		}

		content := tab.Icon + " " + tab.Label

		renderedTabs = append(renderedTabs, style.Render(content))
	}

	separator := lipgloss.NewStyle().Foreground(theme.Surface2).Render("│")
	row := strings.Join(renderedTabs, separator)

	spacer := ""
	spacerWidth := t.Width - lipgloss.Width(row) - 2
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	barStyle := lipgloss.NewStyle().
		Background(theme.Mantle).
		Width(t.Width)

	return barStyle.Render(row + spacer)
}

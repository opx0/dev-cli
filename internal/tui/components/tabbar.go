package components

import (
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

type TabBar struct {
	Tabs       []TabItem
	ActiveTab  int
	Width      int
	ShowMode   bool
	InsertMode bool
	Badges     map[int]int
}

type TabItem struct {
	Icon  string
	Label string
}

func NewTabBar(tabs []TabItem) TabBar {
	return TabBar{
		Tabs:   tabs,
		Badges: make(map[int]int),
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
	t.ShowMode = true
	return t
}

func (t TabBar) SetBadge(tabIdx, count int) TabBar {
	if t.Badges == nil {
		t.Badges = make(map[int]int)
	}
	t.Badges[tabIdx] = count
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

		if count, ok := t.Badges[i]; ok && count > 0 {
			badgeStyle := lipgloss.NewStyle().
				Foreground(theme.Crust).
				Background(theme.Red).
				Bold(true)
			content += " " + badgeStyle.Render(strings.Repeat("•", min(count, 3)))
		}

		renderedTabs = append(renderedTabs, style.Render(content))
	}

	separator := lipgloss.NewStyle().Foreground(theme.Surface2).Render("│")
	row := strings.Join(renderedTabs, separator)

	modeStr := ""
	if t.ShowMode {
		if t.InsertMode {
			modeStr = theme.ModeIndicator.Render(" INSERT ")
		} else {
			modeStr = theme.NormalModeIndicator.Render(" NORMAL ")
		}
	}

	spacer := ""
	spacerWidth := t.Width - lipgloss.Width(row) - lipgloss.Width(modeStr) - 2
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	barStyle := lipgloss.NewStyle().
		Background(theme.Mantle).
		Width(t.Width)

	return barStyle.Render(row + spacer + modeStr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

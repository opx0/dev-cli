package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	crust    = "#11111b"
	mauve    = "#cba6f7"
	red      = "#f38ba8"
	green    = "#a6e3a1"
	overlay0 = "#6c7086"
	surface2 = "#585b70"
	lavender = "#b4befe"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(crust)).
			Background(lipgloss.Color(mauve)).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(surface2)).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(mauve)).
				Padding(0, 1)

	insertModeStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(green)).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(lavender))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(green))

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(red))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay0))

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(mauve)).
			Bold(true)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder())

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(mauve)).
			Foreground(lipgloss.Color(mauve)).
			Bold(true)
)

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.state == StateLoading {
		return m.viewLoading()
	}

	return m.viewMain()
}

func (m Model) viewLoading() string {

	title := titleStyle.Render("dev-cli")
	status := fmt.Sprintf("%s Checking Docker daemon...", m.spinner.View())
	box := panelStyle.Padding(1, 2).Render(status)
	return fmt.Sprintf("\n%s\n\n%s\n", title, box)
}

func (m Model) viewMain() string {
	tabBar := m.renderTabBar()

	var content string
	switch m.activeTab {
	case TabDashboard:
		content = m.renderDashboardTab()
	case TabMonitor:
		content = m.renderMonitorTab()
	case TabAssist:
		content = m.renderAssistTab()
	case TabHistory:
		content = m.renderHistoryTab()
	default:
		content = m.renderDashboardTab()
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

func (m Model) renderTabBar() string {
	tabs := []string{"⊞ Dashboard", "~ Monitor", "? Assist", "↺ History"}
	var renderedTabs []string

	for i, t := range tabs {
		style := tabStyle
		if Tab(i) == m.activeTab {
			style = activeTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	return row + "\n"
}

func (m Model) renderTerminalPanel(width int) string {
	terminalHeight := m.height - 6

	style := panelStyle.Width(width).Height(terminalHeight)

	if m.focus == FocusMain {
		if m.mode == ModeInsert {
			style = insertModeStyle.Width(width).Height(terminalHeight)
		} else {
			style = focusedPanelStyle.Width(width).Height(terminalHeight)
		}
	}

	header := headerStyle.Render(" >_ Terminal")
	var content strings.Builder
	content.WriteString(header + "\n")

	cwdDisplay := m.cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}
	content.WriteString(dimStyle.Render("  "+cwdDisplay) + "\n\n")

	content.WriteString(m.viewport.View())

	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render("❯ "))
	content.WriteString(m.input.View())

	return style.Render(content.String())
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderMonitorTab() string {
	sidebarWidth := 30
	logWidth := m.width - sidebarWidth - 4

	if logWidth < 40 {
		logWidth = 40
	}

	sidebar := m.renderContainerList(sidebarWidth)
	logs := m.renderLogViewport(logWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, logs)
}

func (m Model) renderContainerList(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusSidebar {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" ⬢ Containers")
	var content strings.Builder
	content.WriteString(header + "\n\n")

	if !m.dockerHealth.Available {
		content.WriteString(dimStyle.Render("  Docker unavailable"))
		return style.Render(content.String())
	}

	for i, c := range m.dockerHealth.Containers {
		cursor := " "
		if m.monitorCursor == i {
			cursor = ">"
		}

		name := c.Name
		if len(name) > width-10 {
			name = name[:width-10] + "..."
		}

		stateStyle := runningStyle
		if c.State != "running" {
			stateStyle = stoppedStyle
		}

		line := fmt.Sprintf("%s %s %s", cursor, stateStyle.Render("•"), name)

		if m.monitorCursor == i {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(mauve)).Render(fmt.Sprintf("%s %s %s", cursor, stateStyle.Render("•"), name))
		}

		content.WriteString(line + "\n")
	}

	// Add help tip
	content.WriteString("\n")
	content.WriteString(dimStyle.Render("  [?] RCA Analysis\n"))

	return style.Render(content.String())
}

func (m Model) renderLogViewport(width int) string {
	style := panelStyle.Width(width).Height(m.height - 6)
	if m.focus == FocusMain {
		style = focusedPanelStyle.Width(width).Height(m.height - 6)
	}

	header := headerStyle.Render(" ≡ Logs")
	// If a container is selected, show its name in header
	if m.dockerHealth.Available && len(m.dockerHealth.Containers) > 0 {
		if m.monitorCursor >= 0 && m.monitorCursor < len(m.dockerHealth.Containers) {
			header += dimStyle.Render(" (" + m.dockerHealth.Containers[m.monitorCursor].Name + ")")
		}
	}

	return style.Render(header + "\n" + m.viewport.View())
}

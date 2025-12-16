package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHistoryTab() string {
	sidebarWidth := 30
	detailsWidth := m.width - sidebarWidth - 4

	if detailsWidth < 40 {
		detailsWidth = 40
	}

	sidebar := m.renderHistoryList(sidebarWidth)
	details := m.renderHistoryDetails(detailsWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, details)
}

func (m Model) renderHistoryList(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusSidebar {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" ↺ History")
	var content strings.Builder
	content.WriteString(header + "\n\n")

	if len(m.commandHistory) == 0 {
		content.WriteString(dimStyle.Render("  No history recorded"))
	} else {
		for i, item := range m.commandHistory {
			cursor := " "
			if m.historyCursor == i {
				cursor = ">"
			}
			// Truncate
			disp := item.Command
			if len(disp) > width-5 {
				disp = disp[:width-5] + "..."
			}

			line := fmt.Sprintf("%s %s", cursor, disp)
			if m.historyCursor == i {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(mauve)).Render(line)
			}
			content.WriteString(line + "\n")
		}
	}

	return style.Render(content.String())
}

func (m Model) renderHistoryDetails(width int) string {
	detailsHeight := m.height - 6
	style := panelStyle.Width(width).Height(detailsHeight)
	if m.focus == FocusMain {
		style = focusedPanelStyle.Width(width).Height(detailsHeight)
	}

	header := headerStyle.Render(" i Details")
	content := header + "\n"

	if len(m.commandHistory) > 0 && m.historyCursor >= 0 && m.historyCursor < len(m.commandHistory) {
		item := m.commandHistory[m.historyCursor]
		content += fmt.Sprintf("\nCommand: %s\n", item.Command)
		content += fmt.Sprintf("Time:    %s\n", item.Timestamp.Format("15:04:05 Mon Jan 02"))
		content += fmt.Sprintf("Dir:     %s\n", item.Directory)
		content += fmt.Sprintf("Dur:     %dms\n", item.DurationMs)
		content += fmt.Sprintf("Exit:    %d\n\n", item.ExitCode)

		// If we had output in details, show it?
		// details is JSON, we can just show raw details or try to parse 'output'
		// For now simple display
		content += dimStyle.Render("DetailsJSON: " + item.Details + "\n")
	} else {
		content += "\nNo command selected"
	}

	return style.Render(content)
}

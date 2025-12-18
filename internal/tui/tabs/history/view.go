package history

import (
	"fmt"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {

	sidebarWidth := 40
	if m.width < 100 {
		sidebarWidth = m.width / 3
	}
	if sidebarWidth < 25 {
		sidebarWidth = 25
	}

	detailsWidth := m.width - sidebarWidth - 4
	panelHeight := m.height - 4

	if detailsWidth < 30 {
		detailsWidth = 30
	}
	if panelHeight < 10 {
		panelHeight = 10
	}

	sidebar := m.renderHistoryList(sidebarWidth, panelHeight)
	details := m.renderDetailsPanel(detailsWidth, panelHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, details)
}

func (m Model) renderHistoryList(width, height int) string {

	borderColor := theme.Surface2
	if m.focus == FocusSidebar {
		borderColor = theme.Mauve
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	countStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	header := headerStyle.Render(" ↺ History")
	if len(m.history) > 0 {
		header += countStyle.Render(" " + formatCount(m.list.Index()+1, len(m.history)))
	}

	listContent := m.list.View()

	content := header + "\n" + listContent

	return panelStyle.Render(content)
}

func (m Model) renderDetailsPanel(width, height int) string {

	borderColor := theme.Surface2
	if m.focus == FocusMain {
		borderColor = theme.Mauve
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	header := headerStyle.Render(" ≡ Details")

	content := header + "\n" + m.viewport.View()

	return panelStyle.Render(content)
}

func formatCount(current, total int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Render(" [" + itoa(current) + "/" + itoa(total) + "]")
}

func itoa(i int) string {
	return lipgloss.NewStyle().Render(fmt.Sprintf("%d", i))
}

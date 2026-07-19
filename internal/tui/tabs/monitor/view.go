package monitor

import (
	"fmt"
	"strings"

	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.dockerUnavailable != "" {
		return lipgloss.NewStyle().Foreground(theme.Yellow).Padding(1, 2).
			Render("Docker unavailable: " + m.dockerUnavailable)
	}
	width := m.width - 2
	if width < 10 {
		width = 10
	}
	height := m.height - 2
	if height < 6 {
		height = 6
	}
	if m.width < 70 {
		servicesHeight := height / 3
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderServices(width, servicesHeight),
			m.renderLogs(width, height-servicesHeight-1))
	}
	left := m.width / 3
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderServices(left, height),
		m.renderLogs(m.width-left-2, height))
}

func (m Model) renderServices(width, height int) string {
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Foreground(theme.Lavender).Bold(true).Render("Containers"))
	content.WriteByte('\n')
	if len(m.services) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(theme.Overlay0).Render("No containers"))
	}
	start, end := 0, len(m.services)
	maxItems := height - 3
	if maxItems < 1 {
		maxItems = 1
	}
	if end > maxItems {
		start = m.selected - maxItems/2
		if start < 0 {
			start = 0
		}
		end = start + maxItems
		if end > len(m.services) {
			end = len(m.services)
			start = end - maxItems
		}
	}
	for index := start; index < end; index++ {
		service := m.services[index]
		status, color := "○", theme.Red
		if service.State == "running" {
			status, color = "●", theme.Green
		}
		line := fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(color).Render(status), service.Name)
		if index == m.selected {
			line = lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Lavender).Bold(true).Width(width - 4).Render(line)
		}
		content.WriteString(line + "\n")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.Surface2).
		Width(width).Height(height).Padding(0, 1).Render(strings.TrimRight(content.String(), "\n"))
}

func (m Model) renderLogs(width, height int) string {
	header := "Logs"
	if selected := m.SelectedService(); selected != nil {
		header += " (" + selected.Name + ")"
	}
	if m.logLevelFilter != "" {
		header += " [" + m.logLevelFilter + "]"
	}

	lines := m.filteredLogs()
	maxLines := height - 3
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	if len(lines) == 0 {
		lines = []string{"No logs available"}
	}
	for i := range lines {
		limit := width - 4
		if limit > 1 && len([]rune(lines[i])) > limit {
			lines[i] = string([]rune(lines[i])[:limit-1]) + "…"
		}
		lines[i] = components.NewLogLine(lines[i]).Render()
	}
	content := lipgloss.NewStyle().Foreground(theme.Lavender).Bold(true).Render(header) + "\n" + strings.Join(lines, "\n")
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.Surface2).
		Width(width).Height(height).Padding(0, 1).Render(content)
}

func (m Model) filteredLogs() []string {
	if m.logLevelFilter == "" {
		return append([]string(nil), m.logLines...)
	}
	var lines []string
	for _, line := range m.logLines {
		upper := strings.ToUpper(line)
		if strings.Contains(upper, m.logLevelFilter) ||
			(m.logLevelFilter == "WARN" && strings.Contains(upper, "ERROR")) ||
			(m.logLevelFilter == "INFO" && (strings.Contains(upper, "WARN") || strings.Contains(upper, "ERROR"))) {
			lines = append(lines, line)
		}
	}
	return lines
}

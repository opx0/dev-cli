package monitor

import (
	"fmt"
	"strings"

	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func (m Model) View() string {
	listWidth := 28
	if m.width < 100 {
		listWidth = 24
	}

	logWidth := m.width - listWidth - 4
	if logWidth < 40 {
		logWidth = 40
	}

	panelHeight := m.height - 4
	statsHeight := 8
	listHeight := panelHeight - statsHeight - 1

	if listHeight < 10 {
		listHeight = 10
	}

	containerList := m.renderContainerList(listWidth, listHeight)
	statsPanel := m.renderStatsPanel(listWidth, statsHeight)
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, containerList, statsPanel)

	logsPanel := m.renderLogViewport(logWidth, panelHeight)

	if m.showingActions {
		menu := m.renderActionMenu()
		return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, logsPanel) + "\n" + menu
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, logsPanel)
}

func (m Model) renderContainerList(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusList {
		borderColor = theme.Mauve
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		MaxHeight(height)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	countStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	containerCount := len(m.dockerHealth.Containers)
	posIndicator := ""
	if containerCount > 0 {
		posIndicator = countStyle.Render(fmt.Sprintf(" [%d/%d]", m.containerCursor+1, containerCount))
	}

	header := headerStyle.Render("⬢ Containers") + posIndicator

	var content strings.Builder
	content.WriteString(header + "\n")

	if !m.dockerHealth.Available {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.Red).
			Padding(1).
			Render("✗ Docker unavailable"))
	} else if containerCount == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Padding(1).
			Render("No containers"))
	} else {
		rows := make([][]string, containerCount)
		for i, c := range m.dockerHealth.Containers {
			status := lipgloss.NewStyle().Foreground(theme.Green).Render("●")
			if c.State != "running" {
				status = lipgloss.NewStyle().Foreground(theme.Red).Render("○")
			}

			name := c.Name
			maxWidth := width - 10
			if len(name) > maxWidth && maxWidth > 0 {
				name = name[:maxWidth-1] + "…"
			}

			cursor := " "
			if i == m.containerCursor {
				cursor = "›"
			}

			rows[i] = []string{cursor, status, name}
		}

		t := table.New().
			Border(lipgloss.HiddenBorder()).
			Width(width - 4).
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				baseStyle := lipgloss.NewStyle()
				if row == m.containerCursor {
					baseStyle = baseStyle.Background(theme.Surface1).Bold(true)
				}
				switch col {
				case 0:
					return baseStyle.Foreground(theme.Mauve).Width(2)
				case 1:
					return baseStyle.Width(2)
				case 2:
					if row == m.containerCursor {
						return baseStyle.Foreground(theme.Lavender)
					}
					return baseStyle.Foreground(theme.Text)
				}
				return baseStyle
			})

		content.WriteString(t.Render())
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderStatsPanel(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusStats {
		borderColor = theme.Mauve
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		MaxHeight(height)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	var content strings.Builder
	content.WriteString(headerStyle.Render("▣ Stats") + "\n")

	stats := m.GetSelectedContainerStats()

	labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Width(4)
	content.WriteString(labelStyle.Render("CPU "))

	sparkWidth := width - 12
	if sparkWidth < 5 {
		sparkWidth = 5
	}

	if len(stats.CPUHistory) > 0 {
		sparkline := components.NewSparkline(stats.CPUHistory, 100).
			SetWidth(sparkWidth).
			SetShowValue(true)
		content.WriteString(sparkline.Render())
	} else {
		placeholder := lipgloss.NewStyle().Foreground(theme.Surface1).Render(strings.Repeat("░", sparkWidth))
		content.WriteString(placeholder)
	}
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("MEM "))
	if stats.MemTotal > 0 {
		memBar := components.NewProgressBar(stats.MemUsed, stats.MemTotal).
			SetWidth(sparkWidth)
		content.WriteString(memBar.Render())
	} else {
		placeholder := lipgloss.NewStyle().Foreground(theme.Surface1).Render(strings.Repeat("░", sparkWidth))
		content.WriteString(placeholder)
	}
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("NET "))
	netStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	if stats.NetIn > 0 || stats.NetOut > 0 {
		netStr := fmt.Sprintf("↑%s ↓%s", formatBytes(stats.NetOut), formatBytes(stats.NetIn))
		content.WriteString(netStyle.Render(netStr))
	} else {
		content.WriteString(netStyle.Render("↑0B ↓0B"))
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderLogViewport(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusLogs {
		borderColor = theme.Mauve
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		MaxHeight(height).
		MaxWidth(width)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	header := headerStyle.Render("≡ Logs")
	if m.dockerHealth.Available && len(m.dockerHealth.Containers) > 0 {
		if m.containerCursor >= 0 && m.containerCursor < len(m.dockerHealth.Containers) {
			containerName := m.dockerHealth.Containers[m.containerCursor].Name
			if len(containerName) > 15 {
				containerName = containerName[:12] + "…"
			}
			header += dimStyle.Render(" (" + containerName + ")")
		}
	}

	if m.followMode {
		followBadge := lipgloss.NewStyle().
			Background(theme.Green).
			Foreground(theme.Crust).
			Padding(0, 1).
			Render("F")
		header += " " + followBadge
	}

	if m.logLevelFilter != "" {
		filterBadge := lipgloss.NewStyle().
			Background(theme.Surface0).
			Foreground(theme.Text).
			Padding(0, 1).
			Render(m.logLevelFilter[:1])
		header += " " + filterBadge
	}

	contentWidth := width - 6
	if contentWidth < 20 {
		contentWidth = 20
	}

	contentHeight := height - 4
	if contentHeight < 5 {
		contentHeight = 5
	}

	var displayLines []string
	if len(m.logLines) > 0 {
		filteredLines := m.filterLogLines()

		startIdx := 0
		if len(filteredLines) > contentHeight {
			startIdx = len(filteredLines) - contentHeight
		}
		visibleLines := filteredLines[startIdx:]

		for _, line := range visibleLines {
			truncatedLine := truncateLine(line, contentWidth)
			logLine := components.NewLogLine(truncatedLine)
			displayLines = append(displayLines, logLine.Render())
		}
	} else {
		displayLines = append(displayLines, dimStyle.Render("No logs available"))
		displayLines = append(displayLines, dimStyle.Render("Select a container to view logs"))
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(header + "\n")
	contentBuilder.WriteString(strings.Join(displayLines, "\n"))

	return panelStyle.Render(contentBuilder.String())
}

func truncateLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	runes := []rune(line)
	if len(runes) <= maxWidth {
		return line
	}

	return string(runes[:maxWidth-1]) + "…"
}

func (m Model) filterLogLines() []string {
	if m.logLevelFilter == "" {
		return m.logLines
	}

	var filtered []string
	for _, line := range m.logLines {
		upperLine := strings.ToUpper(line)
		switch m.logLevelFilter {
		case "ERROR":
			if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "ERR") {
				filtered = append(filtered, line)
			}
		case "WARN":
			if strings.Contains(upperLine, "WARN") || strings.Contains(upperLine, "WARNING") ||
				strings.Contains(upperLine, "ERROR") {
				filtered = append(filtered, line)
			}
		case "INFO":
			if strings.Contains(upperLine, "INFO") || strings.Contains(upperLine, "WARN") ||
				strings.Contains(upperLine, "ERROR") {
				filtered = append(filtered, line)
			}
		}
	}
	return filtered
}

func (m Model) renderActionMenu() string {
	container := m.SelectedContainer()
	title := "Actions"
	if container != nil {
		title = container.Name
	}

	items := []components.ActionMenuItem{
		{Key: "s", Label: "shell"},
		{Key: "l", Label: "logs (follow)"},
		{Key: "r", Label: "restart"},
		{Key: "x", Label: "stop"},
		{Key: "i", Label: "inspect"},
	}

	menu := components.NewActionMenu(title, items...).
		SetSelected(m.actionMenuIndex).
		SetWidth(24)

	return menu.Render()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

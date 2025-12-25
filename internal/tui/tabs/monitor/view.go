package monitor

import (
	"fmt"
	"strings"

	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	// Left sidebar width
	sidebarWidth := 28
	if m.width < 100 {
		sidebarWidth = 24
	}

	logWidth := m.width - sidebarWidth - 4
	if logWidth < 40 {
		logWidth = 40
	}

	panelHeight := m.height - 4

	// Calculate heights for left panels
	servicesHeight := (panelHeight - 8) / 2
	imagesHeight := (panelHeight - 8) / 2
	statsHeight := 6

	if servicesHeight < 5 {
		servicesHeight = 5
	}
	if imagesHeight < 5 {
		imagesHeight = 5
	}

	// Render left column panels
	servicesPanel := m.renderServicesPanel(sidebarWidth, servicesHeight)
	imagesPanel := m.renderImagesPanel(sidebarWidth, imagesHeight)
	statsPanel := m.renderStatsPanel(sidebarWidth, statsHeight)

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, servicesPanel, imagesPanel, statsPanel)

	// Render logs panel
	logsPanel := m.renderLogsPanel(logWidth, panelHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, logsPanel)
}

func (m Model) renderServicesPanel(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusServices {
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

	header := headerStyle.Render("⬢ Services")
	if len(m.services) > 0 {
		header += countStyle.Render(fmt.Sprintf(" [%d]", len(m.services)))
	}

	var content strings.Builder
	content.WriteString(header + "\n")

	if len(m.services) == 0 {
		noItems := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Render("No services running")
		content.WriteString(noItems)
	} else {
		content.WriteString(m.servicesList.View())
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderImagesPanel(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusImages {
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

	header := headerStyle.Render("📦 Images")
	if len(m.images) > 0 {
		header += countStyle.Render(fmt.Sprintf(" [%d]", len(m.images)))
	}

	var content strings.Builder
	content.WriteString(header + "\n")

	if len(m.images) == 0 {
		noItems := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Render("No images")
		content.WriteString(noItems)
	} else {
		content.WriteString(m.imagesList.View())
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

	stats := m.GetSelectedServiceStats()
	labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Width(4)

	// CPU sparkline
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

	// Memory bar
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

	// Network
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

func (m Model) renderLogsPanel(width, height int) string {
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

	// Show selected service name
	if svc := m.SelectedService(); svc != nil {
		serviceName := svc.Name
		if len(serviceName) > 15 {
			serviceName = serviceName[:12] + "…"
		}
		header += dimStyle.Render(" (" + serviceName + ")")
	}

	// Recording indicator
	if m.isRecording {
		recBadge := lipgloss.NewStyle().
			Background(theme.Red).
			Foreground(theme.Crust).
			Bold(true).
			Padding(0, 1).
			Render("REC")
		header += " " + recBadge
	}

	// Follow mode indicator
	if m.followMode {
		followBadge := lipgloss.NewStyle().
			Background(theme.Green).
			Foreground(theme.Crust).
			Padding(0, 1).
			Render("F")
		header += " " + followBadge
	}

	// Log level filter indicator
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
		displayLines = append(displayLines, dimStyle.Render("Select a service to view logs"))
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(header + "\n")
	contentBuilder.WriteString(strings.Join(displayLines, "\n"))

	return panelStyle.Render(contentBuilder.String())
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

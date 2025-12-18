package monitor

import (
	"fmt"
	"strings"

	"dev-cli/internal/textutil"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func (m Model) View() string {
	sidebarWidth := 32
	if m.width < 100 {
		sidebarWidth = 28
	}
	logWidth := m.width - sidebarWidth - 4

	if logWidth < 40 {
		logWidth = 40
	}

	panelHeight := m.height - 4
	if panelHeight < 10 {
		panelHeight = 10
	}

	sidebar := m.renderContainerList(sidebarWidth, panelHeight)
	logs := m.renderLogViewport(logWidth, panelHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, logs)
}

func (m Model) renderContainerList(width, height int) string {
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
			Foreground(theme.Overlay0).
			Padding(1).
			Render("Docker unavailable"))
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
			maxWidth := width - 8
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

func (m Model) renderLogViewport(width, height int) string {
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

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	header := headerStyle.Render("≡ Logs")
	if m.dockerHealth.Available && len(m.dockerHealth.Containers) > 0 {
		if m.containerCursor >= 0 && m.containerCursor < len(m.dockerHealth.Containers) {
			containerName := m.dockerHealth.Containers[m.containerCursor].Name
			header += dimStyle.Render(" (" + containerName + ")")
		}
	}

	if m.wrapMode {
		header += " " + lipgloss.NewStyle().
			Background(theme.Surface0).
			Foreground(theme.Text).
			Padding(0, 1).
			Render("wrap")
	} else if m.horizontalOffset > 0 {
		header += dimStyle.Render(fmt.Sprintf(" +%d", m.horizontalOffset))
	}

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	var displayContent string
	if len(m.logLines) > 0 {
		processedLines := textutil.ProcessLinesForViewport(m.logLines, contentWidth, m.horizontalOffset, m.wrapMode)
		displayContent = strings.Join(processedLines, "\n")
	} else {
		displayContent = m.viewport.View()
	}

	var content strings.Builder
	content.WriteString(header + "\n")
	content.WriteString(displayContent)

	return panelStyle.Render(content.String())
}

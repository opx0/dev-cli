package dashboard

import (
	"fmt"
	"os"
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func (m Model) View() string {
	sidebarWidth := 32
	if m.width < 100 {
		sidebarWidth = 28
	}
	terminalWidth := m.width - sidebarWidth - 4

	if terminalWidth < 40 {
		terminalWidth = 40
	}

	panelHeight := m.height - 4
	if panelHeight < 10 {
		panelHeight = 10
	}

	sidebar := m.renderSidebar(sidebarWidth, panelHeight)
	terminal := m.renderTerminal(terminalWidth, panelHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, terminal)
}

func (m Model) renderSidebar(width, height int) string {
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

	var content strings.Builder

	content.WriteString(headerStyle.Render("⬢ Docker") + "\n")
	content.WriteString(m.renderDockerCard() + "\n")

	content.WriteString(headerStyle.Render("≡ Services") + "\n")
	content.WriteString(m.renderServiceTable(width-4) + "\n")

	if m.gpuStats.Available {
		content.WriteString(headerStyle.Render("∿ GPU VRAM") + "\n")
		content.WriteString(m.renderGPUCard(width-4) + "\n")
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderDockerCard() string {
	if !m.dockerHealth.Available {
		return lipgloss.NewStyle().
			Foreground(theme.Red).
			Padding(0, 1).
			Render("✗ Docker unavailable")
	}

	running := 0
	stopped := 0
	for _, c := range m.dockerHealth.Containers {
		if c.State == "running" {
			running++
		} else {
			stopped++
		}
	}

	statusStyle := lipgloss.NewStyle().Foreground(theme.Green)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	return fmt.Sprintf(" %s v%s\n %s%s",
		statusStyle.Render("●"),
		dimStyle.Render(m.dockerHealth.Version),
		lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("%d running", running)),
		dimStyle.Render(fmt.Sprintf(", %d stopped", stopped)))
}

func (m Model) renderServiceTable(width int) string {
	if len(m.serviceHealth) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Padding(0, 1).
			Render("Checking...")
	}

	rows := make([][]string, len(m.serviceHealth))
	for i, s := range m.serviceHealth {
		status := lipgloss.NewStyle().Foreground(theme.Green).Render("●")
		if !s.Available {
			status = lipgloss.NewStyle().Foreground(theme.Red).Render("○")
		}
		rows[i] = []string{status, s.Name, fmt.Sprintf(":%d", s.Port)}
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(theme.Surface2)).
		Width(width).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return lipgloss.NewStyle().Width(2)
			case 1:
				return lipgloss.NewStyle().Foreground(theme.Text).Width(width - 12)
			case 2:
				return lipgloss.NewStyle().Foreground(theme.Overlay0).Width(6)
			}
			return lipgloss.NewStyle()
		})

	return t.Render()
}

func (m Model) renderGPUCard(width int) string {
	pct := m.gpuStats.UtilizationPct
	barWidth := width - 10
	if barWidth < 5 {
		barWidth = 5
	}

	filled := (pct * barWidth) / 100
	var bar strings.Builder

	for i := 0; i < barWidth; i++ {
		var char string
		if i < filled {
			char = "█"
		} else {
			char = "░"
		}

		var color lipgloss.Color
		ratio := float64(i) / float64(barWidth)
		if ratio < 0.5 {
			color = theme.Green
		} else if ratio < 0.75 {
			color = theme.Yellow
		} else {
			color = theme.Red
		}

		if i < filled {
			bar.WriteString(lipgloss.NewStyle().Foreground(color).Render(char))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(theme.Surface1).Render(char))
		}
	}

	pctStyle := lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	if pct > 80 {
		pctStyle = pctStyle.Foreground(theme.Red)
	}

	memStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	memStr := memStyle.Render(fmt.Sprintf("%d/%d MB", m.gpuStats.UsedMemoryMB, m.gpuStats.TotalMemoryMB))

	return fmt.Sprintf(" %s %s\n %s", bar.String(), pctStyle.Render(fmt.Sprintf("%d%%", pct)), memStr)
}

func (m Model) renderTerminal(width, height int) string {
	borderColor := theme.Surface2
	if m.focus == FocusMain {
		if m.insertMode {
			borderColor = theme.Green
		} else {
			borderColor = theme.Mauve
		}
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	cwdStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)

	cwdDisplay := m.cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}

	var content strings.Builder
	content.WriteString(headerStyle.Render(">_ Terminal") + " " + cwdStyle.Render(cwdDisplay) + "\n\n")

	content.WriteString(m.viewport.View())

	promptStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)
	content.WriteString("\n" + promptStyle.Render("❯ ") + m.input.View())

	return panelStyle.Render(content.String())
}

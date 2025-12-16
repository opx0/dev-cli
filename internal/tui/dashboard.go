package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboardTab() string {
	sidebarWidth := 30 // Increased for service health
	terminalWidth := m.width - sidebarWidth - 4

	if terminalWidth < 40 {
		terminalWidth = 40
	}

	// In Dashboard: Left = Status + Services + Recent, Right = "Quick Actions" or similar (or keeping terminal for now)
	// User says: "Tab 1: Dashboard... The 'Heads-Up Display'... Layout: Left (Context), Right (Action)"
	// Right Panel in screenshot is Terminal.
	// "Recent Alerts: A list of last 3 crashed containers or failed commands." -> Left Panel?
	// "Service Health: A quick Red/Green list" -> Left Panel
	// "Quick Actions: A list of frequently used scripts" -> Left Panel? Or maybe Right Panel if it's interactive?

	// Let's stick to the current split:
	// Left: Status (Docker, GPU), Services, Alerts.
	// Right: Terminal (General purpose manual interaction).

	sidebar := m.renderDashboardSidebar(sidebarWidth)
	mainPanel := m.renderTerminalPanel(terminalWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainPanel)
}

func (m Model) renderDashboardSidebar(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusSidebar {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" ⌘ Mission Control")
	var content strings.Builder
	content.WriteString(header + "\n\n")

	// 1. Docker Health
	content.WriteString(m.renderDockerStatus())
	content.WriteString("\n")

	// 2. Service Health
	content.WriteString(m.renderServiceStatus())
	content.WriteString("\n")

	// 3. GPU Health
	content.WriteString(m.renderGPUStatus())
	content.WriteString("\n")

	// 4. Keybinds (Mini)
	content.WriteString(dimStyle.Render("──────────────\n"))
	content.WriteString(dimStyle.Render(" [1-4] Tabs\n"))
	content.WriteString(dimStyle.Render(" [i]   Terminal\n"))

	return style.Render(content.String())
}

func (m Model) renderDockerStatus() string {
	var sb strings.Builder
	if m.dockerHealth.Available {
		sb.WriteString(fmt.Sprintf(" %s ⬢ Docker %s\n",
			runningStyle.Render("•"),
			dimStyle.Render("v"+m.dockerHealth.Version)))

		running := 0
		stopped := 0
		for _, c := range m.dockerHealth.Containers {
			if c.State == "running" {
				running++
			} else {
				stopped++
			}
		}
		sb.WriteString(fmt.Sprintf("   %d running, %d stopped\n", running, stopped))
	} else {
		sb.WriteString(fmt.Sprintf(" %s Docker unavailable\n", stoppedStyle.Render("x")))
	}
	return sb.String()
}

func (m Model) renderServiceStatus() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render(" ≡ Services") + "\n")

	if len(m.serviceHealth) == 0 {
		sb.WriteString(dimStyle.Render("   Checking...\n"))
		return sb.String()
	}

	for _, s := range m.serviceHealth {
		icon := runningStyle.Render("√")
		if !s.Available {
			icon = stoppedStyle.Render("x")
		}
		sb.WriteString(fmt.Sprintf(" %s %s (%d)\n", icon, s.Name, s.Port))
	}
	return sb.String()
}

func (m Model) renderGPUStatus() string {
	var sb strings.Builder
	if m.gpuStats.Available {
		sb.WriteString(fmt.Sprintf(" %s ~ GPU VRAM\n", runningStyle.Render("~")))
		pct := m.gpuStats.UtilizationPct
		bars := pct / 10 // 10 bars total

		var barStr string
		for i := 0; i < 10; i++ {
			if i < bars {
				barStr += "█"
			} else {
				barStr += "░"
			}
		}

		barStyle := runningStyle
		if pct > 80 {
			barStyle = stoppedStyle
		}

		sb.WriteString(fmt.Sprintf("   %s %d%%\n", barStyle.Render(barStr), pct))
		sb.WriteString(fmt.Sprintf("   %s/%s MB\n",
			fmt.Sprint(m.gpuStats.UsedMemoryMB),
			fmt.Sprint(m.gpuStats.TotalMemoryMB),
		))
	} else {
		// Only show if we actually expect a GPU or have an error
		// For now, minimal output if no GPU found to save space, or show error?
		// User requirement: "Crucial ... dedicated VRAM Monitor"
		if m.gpuStats.Error != nil {
			// sb.WriteString(fmt.Sprintf(" %s No GPU found\n", dimStyle.Render("-")))
		}
	}
	return sb.String()
}

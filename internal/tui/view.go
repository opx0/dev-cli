package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	crust    = "#11111b"
	mantle   = "#181825"
	base     = "#1e1e2e"
	surface0 = "#313244"
	surface1 = "#45475a"
	surface2 = "#585b70"
	overlay0 = "#6c7086"
	overlay1 = "#7f849c"
	subtext0 = "#a6adc8"
	subtext1 = "#bac2de"
	text     = "#cdd6f4"

	mauve    = "#cba6f7"
	red      = "#f38ba8"
	green    = "#a6e3a1"
	teal     = "#94e2d5"
	yellow   = "#f9e2af"
	blue     = "#89b4fa"
	lavender = "#b4befe"
	peach    = "#fab387"
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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay1))
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

	var status string
	if m.dockerHealth.Error != nil {
		status = fmt.Sprintf(
			"%s Checking infrastructure...\n\n%s %s",
			m.spinner.View(),
			lipgloss.NewStyle().Foreground(lipgloss.Color(red)).Bold(true).Render("Docker: ✗"),
			dimStyle.Render("("+m.dockerHealth.Error.Error()+")"),
		)
	} else {
		status = fmt.Sprintf("%s Checking Docker daemon...", m.spinner.View())
	}

	box := panelStyle.Padding(1, 2).Render(status)
	return fmt.Sprintf("\n%s\n\n%s\n", title, box)
}

func (m Model) viewMain() string {
	sidebarWidth := 24
	terminalWidth := m.width - sidebarWidth - 4

	if terminalWidth < 40 {
		terminalWidth = 40
	}

	containerPanel := m.renderContainersPanel(sidebarWidth)
	observabilityPanel := m.renderObservabilityPanel(sidebarWidth)
	keybindsPanel := m.renderKeybindsPanel(sidebarWidth)

	sidebar := lipgloss.JoinVertical(lipgloss.Left,
		containerPanel,
		observabilityPanel,
		keybindsPanel,
	)

	terminal := m.renderTerminalPanel(terminalWidth)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, terminal)

	title := titleStyle.Render("dev-cli")
	return fmt.Sprintf("\n%s\n\n%s\n", title, layout)
}

func (m Model) renderContainersPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusContainers {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render("󰡨 Containers") + " " + keyStyle.Render("[1]")
	var content strings.Builder
	content.WriteString(header + "\n")

	if len(m.dockerHealth.Containers) == 0 {
		content.WriteString(dimStyle.Render("  No containers"))
	} else {
		for i, c := range m.dockerHealth.Containers {
			if i >= 5 {
				content.WriteString(dimStyle.Render(fmt.Sprintf("  +%d more...", len(m.dockerHealth.Containers)-5)))
				break
			}
			icon := runningStyle.Render("●")
			if c.State != "running" {
				icon = stoppedStyle.Render("○")
			}
			name := c.Name
			if len(name) > 15 {
				name = name[:15] + "…"
			}
			content.WriteString(fmt.Sprintf("  %s %s\n", icon, name))
		}
	}

	return style.Render(content.String())
}

func (m Model) renderObservabilityPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusObservability {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render("󰈈 Observability") + " " + keyStyle.Render("[2]")
	var content strings.Builder
	content.WriteString(header + "\n")

	runningCount := 0
	for _, c := range m.dockerHealth.Containers {
		if c.State == "running" {
			runningCount++
		}
	}

	content.WriteString(fmt.Sprintf("  Docker: %s\n", runningStyle.Render("v"+m.dockerHealth.Version)))
	content.WriteString(fmt.Sprintf("  Running: %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render(fmt.Sprintf("%d", runningCount))))
	content.WriteString(fmt.Sprintf("  Total: %s", dimStyle.Render(fmt.Sprintf("%d", len(m.dockerHealth.Containers)))))

	return style.Render(content.String())
}

func (m Model) renderKeybindsPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusKeybinds {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" Keybinds") + " " + keyStyle.Render("[3]")
	var content strings.Builder
	content.WriteString(header + "\n")

	binds := []struct{ key, desc string }{
		{"1-4", "focus panel"},
		{"Tab", "next panel"},
		{"Enter", "run command"},
		{"Ctrl+C", "quit"},
	}

	for _, b := range binds {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			keyStyle.Render(b.key),
			dimStyle.Render(b.desc)))
	}

	return style.Render(content.String())
}

func (m Model) renderTerminalPanel(width int) string {
	style := panelStyle.Width(width).Height(m.height - 8)
	if m.focus == FocusTerminal {
		style = focusedPanelStyle.Width(width).Height(m.height - 8)
	}

	header := headerStyle.Render(" Terminal") + " " + keyStyle.Render("[4]")
	var content strings.Builder
	content.WriteString(header + "\n\n")

	maxHistory := m.height - 14
	if maxHistory < 3 {
		maxHistory = 3
	}

	start := 0
	if len(m.cmdHistory) > maxHistory {
		start = len(m.cmdHistory) - maxHistory
	}

	for _, cmd := range m.cmdHistory[start:] {
		content.WriteString(dimStyle.Render(cmd) + "\n")
	}

	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render("❯ "))
	content.WriteString(m.input.View())

	return style.Render(content.String())
}

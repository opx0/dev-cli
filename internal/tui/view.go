package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ... (Existing colors remain the same) ...
const (
	crust    = "#11111b"
	mantle   = "#181825"
	mauve    = "#cba6f7"
	red      = "#f38ba8"
	green    = "#a6e3a1"
	overlay0 = "#6c7086"
	overlay1 = "#7f849c"
	surface2 = "#585b70"
	lavender = "#b4befe"
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

	// NEW: Style for Insert Mode focus
	insertModeStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(green)).
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
)

// ... (View and ViewLoading remain similar) ...

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
	// ... same as before ...
	title := titleStyle.Render("dev-cli")
	status := fmt.Sprintf("%s Checking Docker daemon...", m.spinner.View())
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

	// Terminal fills full height to match sidebar
	terminal := m.renderTerminalPanel(terminalWidth)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, terminal)

	// Render Title with Mode Indicator
	modeStr := "NORMAL"
	modeColor := lipgloss.Color(mauve)
	if m.mode == ModeInsert {
		modeStr = "INSERT"
		modeColor = lipgloss.Color(green)
	}

	title := titleStyle.Background(modeColor).Render("dev-cli") + " " +
		lipgloss.NewStyle().Foreground(modeColor).Bold(true).Render(modeStr)

	return fmt.Sprintf("\n%s\n\n%s\n", title, layout)
}

// ... (renderContainersPanel and renderObservabilityPanel remain same) ...
func (m Model) renderContainersPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusContainers && m.mode == ModeNormal {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render("󰡨 Containers")
	var content strings.Builder
	content.WriteString(header + "\n")

	if len(m.dockerHealth.Containers) == 0 {
		content.WriteString(dimStyle.Render("  No containers"))
	} else {
		for _, c := range m.dockerHealth.Containers {
			// Truncate name to fit panel
			name := c.Name
			maxLen := width - 6
			if len(name) > maxLen {
				name = name[:maxLen-2] + "…"
			}

			// Status indicator
			var statusIcon string
			if c.State == "running" {
				statusIcon = runningStyle.Render("●")
			} else {
				statusIcon = stoppedStyle.Render("○")
			}

			content.WriteString(fmt.Sprintf("  %s %s\n", statusIcon, name))
		}
	}

	return style.Render(content.String())
}

func (m Model) renderObservabilityPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusObservability && m.mode == ModeNormal {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" Observability")
	var content strings.Builder
	content.WriteString(header + "\n")

	// Docker status
	if m.dockerHealth.Available {
		content.WriteString(fmt.Sprintf("  %s Docker %s\n",
			runningStyle.Render("✓"),
			dimStyle.Render("v"+m.dockerHealth.Version)))

		// Container counts
		running := 0
		stopped := 0
		for _, c := range m.dockerHealth.Containers {
			if c.State == "running" {
				running++
			} else {
				stopped++
			}
		}
		content.WriteString(fmt.Sprintf("  %s %d running\n",
			runningStyle.Render("●"), running))
		content.WriteString(fmt.Sprintf("  %s %d stopped\n",
			stoppedStyle.Render("○"), stopped))
	} else {
		content.WriteString(fmt.Sprintf("  %s Docker unavailable\n",
			stoppedStyle.Render("✗")))
	}

	return style.Render(content.String())
}

func (m Model) renderKeybindsPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusKeybinds && m.mode == ModeNormal {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" Keybinds")
	var content strings.Builder
	content.WriteString(header + "\n")

	// Dynamic Keybinds based on Mode
	var binds []struct{ key, desc string }
	if m.mode == ModeNormal {
		binds = []struct{ key, desc string }{
			{"j/k", "nav panels"},
			{"1-4", "jump panel"},
			{"↑/↓", "scroll term"},
			{"i", "insert mode"},
			{"q", "quit"},
		}
	} else {
		binds = []struct{ key, desc string }{
			{"Esc", "normal mode"},
			{"Enter", "run command"},
			{"!", "force interact"},
		}
	}

	for _, b := range binds {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			keyStyle.Render(b.key),
			dimStyle.Render(b.desc)))
	}

	return style.Render(content.String())
}

func (m Model) renderTerminalPanel(width int) string {
	// Terminal fills from top to bottom edge
	terminalHeight := m.height - 6

	// Determine Style based on Focus and Mode
	style := panelStyle.Width(width).Height(terminalHeight)

	if m.focus == FocusTerminal {
		if m.mode == ModeInsert {
			style = insertModeStyle.Width(width).Height(terminalHeight)
		} else {
			style = focusedPanelStyle.Width(width).Height(terminalHeight)
		}
	}

	header := headerStyle.Render(" Terminal")
	var content strings.Builder
	content.WriteString(header + "\n")

	// Show current working directory
	cwdDisplay := m.cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}
	content.WriteString(dimStyle.Render("  "+cwdDisplay) + "\n\n")

	// Render Viewport (History)
	content.WriteString(m.viewport.View())

	// Render Input Line
	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render("❯ "))
	content.WriteString(m.input.View())

	return style.Render(content.String())
}

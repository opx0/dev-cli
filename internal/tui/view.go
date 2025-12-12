package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	crust = "#11111b"
	// mantle   = "#181825"
	mauve    = "#cba6f7"
	red      = "#f38ba8"
	green    = "#a6e3a1"
	overlay0 = "#6c7086"
	// overlay1 = "#7f849c"
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

	statusPanel := m.renderStatusPanel(sidebarWidth)
	keybindsPanel := m.renderKeybindsPanel(sidebarWidth)

	sidebar := lipgloss.JoinVertical(lipgloss.Left,
		statusPanel,
		keybindsPanel,
	)

	terminal := m.renderTerminalPanel(terminalWidth)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, terminal)

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

func (m Model) renderStatusPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusStatus && m.mode == ModeNormal {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" Status")
	var content strings.Builder
	content.WriteString(header + "\n")

	// Docker status
	if m.dockerHealth.Available {
		content.WriteString(fmt.Sprintf("  %s Docker %s\n",
			runningStyle.Render("✓"),
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
		content.WriteString(fmt.Sprintf("  %s %d\n",
			runningStyle.Render("●"), running))
		content.WriteString(fmt.Sprintf("  %s %d\n",
			stoppedStyle.Render("○"), stopped))
	} else {
		content.WriteString(fmt.Sprintf("  %s Docker unavailable\n",
			stoppedStyle.Render("✗")))
	}

	content.WriteString("\n") // Spacer

	// Container list
	if len(m.dockerHealth.Containers) == 0 {
		content.WriteString(dimStyle.Render("  No containers"))
	} else {
		for _, c := range m.dockerHealth.Containers {

			name := c.Name
			maxLen := width - 6
			if len(name) > maxLen {
				name = name[:maxLen-2] + "…"
			}

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

func (m Model) renderKeybindsPanel(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusKeybinds && m.mode == ModeNormal {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" Keybinds")
	var content strings.Builder
	content.WriteString(header + "\n")

	var binds []struct{ key, desc string }
	if m.mode == ModeNormal {
		binds = []struct{ key, desc string }{
			{"j/k", "nav panels"},
			{"1-3", "jump panel"},
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

	terminalHeight := m.height - 6

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

	cwdDisplay := m.cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}
	content.WriteString(dimStyle.Render("  "+cwdDisplay) + "\n\n")

	content.WriteString(m.viewport.View())

	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render("❯ "))
	content.WriteString(m.input.View())

	return style.Render(content.String())
}

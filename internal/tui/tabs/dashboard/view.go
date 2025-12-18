package dashboard

import (
	"fmt"
	"os"
	"strings"

	"dev-cli/internal/tui/components"
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	contentWidth := m.width - 2
	if contentWidth < 40 {
		contentWidth = 40
	}

	var content strings.Builder

	content.WriteString(m.renderHeaderBar(contentWidth) + "\n")

	content.WriteString(m.renderOutputArea(contentWidth, m.height-8) + "\n")

	content.WriteString(m.renderInputArea(contentWidth))

	return content.String()
}

func (m Model) renderHeaderBar(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Lavender)

	title := titleStyle.Render("⌘ Command Center")

	cwdStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)

	cwdDisplay := m.cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}

	cwd := cwdStyle.Render(" " + cwdDisplay)

	widgetBar := components.NewHeaderWidgetBar(m.HeaderWidgets()...).SetWidth(width)
	widgetsStr := widgetBar.Render()

	leftSide := title + cwd
	leftWidth := lipgloss.Width(leftSide)
	widgetsWidth := lipgloss.Width(widgetsStr)

	spacerWidth := width - leftWidth - widgetsWidth
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	headerBar := lipgloss.NewStyle().
		Background(theme.Mantle).
		Width(width).
		Padding(0, 1)

	return headerBar.Render(leftSide + spacer + widgetsStr)
}

func (m Model) renderOutputArea(width, height int) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Width(width).
		Height(height)

	if m.insertMode {
		panelStyle = panelStyle.BorderForeground(theme.Green)
	} else {
		panelStyle = panelStyle.BorderForeground(theme.Mauve)
	}

	var content strings.Builder

	if len(m.outputBlocks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Italic(true).
			Padding(2, 2)

		welcomeMsg := `Welcome to dev-cli Command Center

Press 'i' to start typing commands
Use 'j/k' to navigate output blocks
Press '?' for quick actions

Try: docker ps, npm test, git status`

		content.WriteString(emptyStyle.Render(welcomeMsg))
	} else {
		blockWidth := width - 6
		for i, block := range m.outputBlocks {
			blockComp := components.NewOutputBlock(block.Command).
				SetOutput(block.Output).
				SetExitCode(block.ExitCode).
				SetTimestamp(block.Timestamp).
				SetFolded(block.Folded).
				SetSelected(i == m.selectedBlock)

			content.WriteString(blockComp.Render(blockWidth) + "\n\n")
		}
	}

	viewportContent := m.viewport.View()
	if viewportContent != "" && len(m.outputBlocks) == 0 {
		content.WriteString(viewportContent)
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderInputArea(width int) string {
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Width(width).
		Padding(0, 1)

	if m.insertMode {
		inputStyle = inputStyle.BorderForeground(theme.Green)
	}

	promptStyle := theme.Prompt
	prompt := promptStyle.Render("❯ ")

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)

	hint := ""
	if !m.insertMode {
		hint = hintStyle.Render("  [i]nsert [?]actions [j/k]nav")
	} else {
		hint = hintStyle.Render("  [esc] normal mode")
	}

	inputRow := prompt + m.input.View()

	inputWidth := lipgloss.Width(inputRow)
	hintWidth := lipgloss.Width(hint)
	spacerWidth := width - inputWidth - hintWidth - 4
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	return inputStyle.Render(inputRow + spacer + hint)
}

func (m Model) RenderActionMenu() string {
	if !m.showingActions {
		return ""
	}

	var items []components.ActionMenuItem

	if len(m.outputBlocks) > 0 && m.selectedBlock >= 0 {
		block := m.outputBlocks[m.selectedBlock]
		if block.ExitCode != 0 {
			items = append(items, components.ActionMenuItem{Key: "r", Label: "Retry command"})
			items = append(items, components.ActionMenuItem{Key: "e", Label: "Explain error"})
			items = append(items, components.ActionMenuItem{Key: "f", Label: "Fix with AI"})
		}
	}

	if m.dockerHealth.Available && len(m.dockerHealth.Containers) > 0 {
		items = append(items, components.ActionMenuItem{Key: "l", Label: "View logs"})
		items = append(items, components.ActionMenuItem{Key: "s", Label: "Shell into container"})
	}

	items = append(items, components.ActionMenuItem{Key: "c", Label: "Clear screen"})
	items = append(items, components.ActionMenuItem{Key: "h", Label: "View history"})

	menu := components.NewActionMenu("Quick Actions", items...)

	menuStr := menu.Render()
	menuWidth := lipgloss.Width(menuStr)
	menuHeight := lipgloss.Height(menuStr)

	posX := (m.width - menuWidth) / 2
	posY := (m.height - menuHeight) / 2

	return lipgloss.NewStyle().
		MarginLeft(posX).
		MarginTop(posY).
		Render(menuStr)
}

func (m Model) StatusInfo() string {
	var info []string

	if m.dockerHealth.Available {
		running := 0
		for _, c := range m.dockerHealth.Containers {
			if c.State == "running" {
				running++
			}
		}
		info = append(info, fmt.Sprintf("🐳 %d/%d", running, len(m.dockerHealth.Containers)))
	}

	if m.gpuStats.Available {
		info = append(info, fmt.Sprintf("GPU %d%%", m.gpuStats.UtilizationPct))
	}

	if len(m.outputBlocks) > 0 {
		info = append(info, fmt.Sprintf("📋 %d blocks", len(m.outputBlocks)))
	}

	return strings.Join(info, " │ ")
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAssistTab() string {
	sidebarWidth := 20
	chatWidth := m.width - sidebarWidth - 4

	if chatWidth < 40 {
		chatWidth = 40
	}

	sidebar := m.renderAssistSidebar(sidebarWidth)
	chat := m.renderChatPanel(chatWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)
}

func (m Model) renderAssistSidebar(width int) string {
	style := panelStyle.Width(width)
	if m.focus == FocusSidebar {
		style = focusedPanelStyle.Width(width)
	}

	header := headerStyle.Render(" ∞ AI Model")
	var content strings.Builder
	content.WriteString(header + "\n\n")

	// Mode Toggle
	localStyle := dimStyle
	cloudStyle := dimStyle

	if m.aiMode == "local" {
		localStyle = runningStyle
		content.WriteString(fmt.Sprintf("%s Local (Ollama)\n", localStyle.Render("•")))
		content.WriteString("  Cloud (Perplexity)\n")
	} else {
		cloudStyle = runningStyle
		content.WriteString("  Local (Ollama)\n")
		content.WriteString(fmt.Sprintf("%s Cloud (Perplexity)\n", cloudStyle.Render("•")))
	}

	content.WriteString("\n")
	content.WriteString(dimStyle.Render("Ctrl+t: Toggle Mode\n"))

	return style.Render(content.String())
}

func (m Model) renderChatPanel(width int) string {
	chatHeight := m.height - 6
	style := panelStyle.Width(width).Height(chatHeight)
	if m.focus == FocusMain {
		style = focusedPanelStyle.Width(width).Height(chatHeight)
	}

	header := headerStyle.Render(" [...] Chat")

	var content strings.Builder
	content.WriteString(header + "\n")

	content.WriteString(m.viewport.View())

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(green)).Render("❯ "))
	content.WriteString(m.input.View())

	return style.Render(content.String())
}

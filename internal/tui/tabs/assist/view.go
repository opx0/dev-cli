package assist

import (
	"fmt"
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	sidebarWidth := 24
	if m.width < 100 {
		sidebarWidth = 20
	}
	chatWidth := m.width - sidebarWidth - 4

	if chatWidth < 40 {
		chatWidth = 40
	}

	panelHeight := m.height - 4
	if panelHeight < 10 {
		panelHeight = 10
	}

	sidebar := m.renderSidebar(sidebarWidth, panelHeight)
	chat := m.renderChatPanel(chatWidth, panelHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)
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

	activeStyle := lipgloss.NewStyle().
		Foreground(theme.Green).
		Bold(true)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	selectedBg := lipgloss.NewStyle().
		Background(theme.Surface1).
		Width(width-4).
		Padding(0, 1)

	var content strings.Builder
	content.WriteString(headerStyle.Render("∞ AI Mode") + "\n\n")

	localItem := " ● Local (Ollama)"
	cloudItem := " ○ Cloud (Perplexity)"

	if m.aiMode == "local" {
		content.WriteString(selectedBg.Render(activeStyle.Render(localItem)) + "\n")
		content.WriteString(" " + inactiveStyle.Render(cloudItem) + "\n")
	} else {
		content.WriteString(" " + inactiveStyle.Render(localItem) + "\n")
		content.WriteString(selectedBg.Render(activeStyle.Render(cloudItem)) + "\n")
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true).
		Render("  Ctrl+t to toggle"))

	return panelStyle.Render(content.String())
}

func (m Model) renderChatPanel(width, height int) string {
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

	countStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	msgCount := len(m.chatHistory)
	posIndicator := ""
	if msgCount > 0 {
		posIndicator = countStyle.Render(fmt.Sprintf(" (%d msgs)", msgCount))
	}

	header := headerStyle.Render("💬 Chat") + posIndicator

	var content strings.Builder
	content.WriteString(header + "\n\n")

	content.WriteString(m.viewport.View())

	promptStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)
	content.WriteString("\n" + promptStyle.Render("❯ ") + m.input.View())

	return panelStyle.Render(content.String())
}

package assist

import (
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

	// Header bar with AI mode toggle
	content.WriteString(m.renderHeaderBar(contentWidth) + "\n")

	// Chat area
	content.WriteString(m.renderChatArea(contentWidth, m.height-8) + "\n")

	// Input area
	content.WriteString(m.renderInputArea(contentWidth))

	return content.String()
}

func (m Model) renderHeaderBar(width int) string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Lavender)

	title := titleStyle.Render("◇ Assistant")

	// AI mode badge
	var modeBadge string
	if m.aiMode == "local" {
		modeBadge = lipgloss.NewStyle().
			Background(theme.Green).
			Foreground(theme.Crust).
			Padding(0, 1).
			Bold(true).
			Render("ollama ●")
	} else {
		modeBadge = lipgloss.NewStyle().
			Background(theme.Blue).
			Foreground(theme.Crust).
			Padding(0, 1).
			Bold(true).
			Render("cloud ●")
	}

	// Toggle hint
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)
	hint := hintStyle.Render(" [Ctrl+t]")

	// Context badge
	contextBadge := ""
	if m.recentCommands > 0 || m.containerCount > 0 || m.errorCount > 0 {
		ctx := components.NewContextBadge().
			SetCommands(m.recentCommands).
			SetContainers(m.containerCount).
			SetErrors(m.errorCount)
		contextBadge = ctx.Render()
	}

	// Build header
	leftSide := title
	rightSide := contextBadge + "  " + modeBadge + hint

	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)

	spacerWidth := width - leftWidth - rightWidth
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	headerBar := lipgloss.NewStyle().
		Background(theme.Mantle).
		Width(width).
		Padding(0, 1)

	return headerBar.Render(leftSide + spacer + rightSide)
}

func (m Model) renderChatArea(width, height int) string {
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

	if len(m.messages) == 0 {
		// Empty state with suggestions
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Italic(true).
			Padding(2, 2)

		welcomeMsg := `Welcome to dev-cli Assistant

I can help you with:
  • Debugging Docker containers
  • Explaining error messages
  • Generating configurations
  • DevOps best practices

Press 'i' to start chatting

Try: "Why is my container restarting?"
     "Generate a Dockerfile for Node.js"
     "Explain this error: connection refused"`

		content.WriteString(emptyStyle.Render(welcomeMsg))
	} else {
		// Render messages as chat bubbles
		contentWidth := width - 8
		for _, msg := range m.messages {
			content.WriteString(m.renderMessage(msg, contentWidth) + "\n\n")
		}

		// Loading indicator
		if m.isLoading {
			loadingStyle := lipgloss.NewStyle().
				Foreground(theme.Overlay0).
				Italic(true)
			content.WriteString(loadingStyle.Render("  ◌ Thinking..."))
		}
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderMessage(msg ChatMessage, width int) string {
	if msg.Role == "user" {
		return m.renderUserMessage(msg.Content, width)
	}
	return m.renderAssistantMessage(msg.Content, width)
}

func (m Model) renderUserMessage(content string, width int) string {
	// User messages aligned right with bubble style
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true)

	bubbleStyle := lipgloss.NewStyle().
		Background(theme.Surface1).
		Foreground(theme.Text).
		Padding(0, 1).
		MaxWidth(width - 10)

	label := labelStyle.Render("You")
	bubble := bubbleStyle.Render(content)

	// Right align
	labelWidth := lipgloss.Width(label)
	bubbleWidth := lipgloss.Width(bubble)

	maxContentWidth := labelWidth
	if bubbleWidth > maxContentWidth {
		maxContentWidth = bubbleWidth
	}

	padding := width - maxContentWidth - 4
	if padding < 0 {
		padding = 0
	}

	paddedLabel := strings.Repeat(" ", padding) + label
	paddedBubble := strings.Repeat(" ", padding) + bubble

	return paddedLabel + "\n" + paddedBubble
}

func (m Model) renderAssistantMessage(content string, width int) string {
	// Assistant messages aligned left with different style
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	var modeIcon string
	if m.aiMode == "local" {
		modeIcon = "🦙"
	} else {
		modeIcon = "☁️"
	}

	label := labelStyle.Render(modeIcon + " Assistant")

	// Check if content contains code blocks
	if strings.Contains(content, "```") {
		return "  " + label + "\n" + m.renderContentWithCodeBlocks(content, width-4)
	}

	bubbleStyle := lipgloss.NewStyle().
		Background(theme.Surface0).
		Foreground(theme.Text).
		Padding(0, 1).
		MaxWidth(width - 10)

	bubble := bubbleStyle.Render(content)

	return "  " + label + "\n  " + bubble
}

func (m Model) renderContentWithCodeBlocks(content string, width int) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")
	inCodeBlock := false

	textStyle := lipgloss.NewStyle().
		Background(theme.Surface0).
		Foreground(theme.Text).
		Padding(0, 1)

	codeStyle := lipgloss.NewStyle().
		Background(theme.Mantle).
		Foreground(theme.Text).
		Padding(0, 1)

	actionStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)

	var textBuffer strings.Builder
	var codeBuffer strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End code block
				result.WriteString("  " + codeStyle.MaxWidth(width).Render(codeBuffer.String()) + "\n")
				result.WriteString("  " + actionStyle.Render("[Copy]") + " " + actionStyle.Render("[Apply]") + "\n")
				codeBuffer.Reset()
				inCodeBlock = false
			} else {
				// Start code block - flush text buffer
				if textBuffer.Len() > 0 {
					result.WriteString("  " + textStyle.MaxWidth(width).Render(textBuffer.String()) + "\n")
					textBuffer.Reset()
				}
				inCodeBlock = true
			}
		} else {
			if inCodeBlock {
				if codeBuffer.Len() > 0 {
					codeBuffer.WriteString("\n")
				}
				codeBuffer.WriteString(line)
			} else {
				if textBuffer.Len() > 0 {
					textBuffer.WriteString("\n")
				}
				textBuffer.WriteString(line)
			}
		}
	}

	// Flush remaining buffers
	if textBuffer.Len() > 0 {
		result.WriteString("  " + textStyle.MaxWidth(width).Render(textBuffer.String()))
	}

	return result.String()
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

	// Mode hint
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)

	hint := ""
	if !m.insertMode {
		hint = hintStyle.Render("  [i]nsert [Ctrl+t]toggle AI [j/k]scroll")
	} else {
		hint = hintStyle.Render("  [Enter]send [Esc]normal")
	}

	inputRow := prompt + m.input.View()

	// Calculate space for hint
	inputWidth := lipgloss.Width(inputRow)
	hintWidth := lipgloss.Width(hint)
	spacerWidth := width - inputWidth - hintWidth - 4
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	return inputStyle.Render(inputRow + spacer + hint)
}

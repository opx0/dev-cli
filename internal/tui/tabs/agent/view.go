package agent

import (
	"fmt"
	"os"
	"strings"

	"dev-cli/internal/pipeline"
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

	starshipHeight := 0
	if m.StarshipLine() != "" {
		starshipHeight = 1
	}
	blocksHeight := m.height - 8 - starshipHeight

	content.WriteString(m.renderBlocksArea(contentWidth, blocksHeight) + "\n")

	if m.StarshipLine() != "" {
		content.WriteString(m.renderStarshipBar(contentWidth) + "\n")
	}

	content.WriteString(m.renderInputArea(contentWidth))

	return content.String()
}

func (m Model) renderHeaderBar(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Lavender)

	title := titleStyle.Render("◈ Agent")

	cwdStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true)

	cwdDisplay := m.Cwd()
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwdDisplay, home) {
		cwdDisplay = "~" + cwdDisplay[len(home):]
	}
	maxCwdLen := 30
	if len(cwdDisplay) > maxCwdLen {
		cwdDisplay = "..." + cwdDisplay[len(cwdDisplay)-maxCwdLen+3:]
	}
	cwd := cwdStyle.Render(" " + cwdDisplay)

	var widgets []string

	gitStatus := m.GitStatus()
	if gitStatus.IsRepo {
		gitStyle := lipgloss.NewStyle().Foreground(theme.Mauve)
		branchText := "⎇ " + gitStatus.Branch
		changes := gitStatus.Added + gitStatus.Modified + gitStatus.Deleted + gitStatus.Untracked
		if changes > 0 {
			branchText += fmt.Sprintf(" •%d", changes)
		}
		widgets = append(widgets, gitStyle.Render(branchText))
	}

	dockerHealth := m.DockerHealth()
	if dockerHealth.Available {
		running := 0
		for _, c := range dockerHealth.Containers {
			if c.State == "running" {
				running++
			}
		}
		dockerStyle := lipgloss.NewStyle().Foreground(theme.Green)
		widgets = append(widgets, dockerStyle.Render(fmt.Sprintf("🐳 %d", running)))
	}

	gpuStats := m.GPUStats()
	if gpuStats.Available {
		gpuStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		if gpuStats.UtilizationPct > 80 {
			gpuStyle = gpuStyle.Foreground(theme.Red)
		}
		widgets = append(widgets, gpuStyle.Render(fmt.Sprintf("▮ %d%%", gpuStats.UtilizationPct)))
	}

	aiStyle := lipgloss.NewStyle().
		Background(theme.Surface0).
		Foreground(theme.Green).
		Padding(0, 1)
	widgets = append(widgets, aiStyle.Render(m.AIMode()+" ●"))

	widgetStr := strings.Join(widgets, " │ ")

	leftSide := title + cwd
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(widgetStr)

	spacerWidth := width - leftWidth - rightWidth - 2
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	headerBar := lipgloss.NewStyle().
		Background(theme.Mantle).
		Width(width).
		Padding(0, 1)

	return headerBar.Render(leftSide + spacer + widgetStr)
}

func (m Model) renderBlocksArea(width, height int) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Width(width).
		Height(height)

	if m.insertMode {
		panelStyle = panelStyle.BorderForeground(theme.Green)
	} else if m.selectedBlock >= 0 {
		panelStyle = panelStyle.BorderForeground(theme.Mauve)
	}

	var content strings.Builder

	blocks := m.Blocks()
	if len(blocks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Padding(2, 2)

		welcomeMsg := `  Welcome to dev-cli Agent

  Commands:
    Type any shell command and press Enter
    Your zsh aliases and functions work here!

  AI Queries:
    ? how to fix permission denied
    @fix      - Fix last error
    @explain  - Explain last command

  Navigation:
    j/k  - Navigate blocks
    z    - Fold/unfold block
    i    - Insert mode
`
		content.WriteString(emptyStyle.Render(welcomeMsg))
	} else {
		blockWidth := width - 6
		for i, block := range blocks {
			content.WriteString(m.renderBlock(block, i, blockWidth) + "\n")
		}

		if m.isExecuting {
			execStyle := lipgloss.NewStyle().
				Foreground(theme.Yellow).
				Italic(true)
			content.WriteString(execStyle.Render("  ◌ Executing..."))
		}
	}

	return panelStyle.Render(content.String())
}

func (m Model) renderBlock(block pipeline.Block, index int, width int) string {
	isSelected := index == m.selectedBlock

	var borderStyle lipgloss.Style
	if isSelected {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "▐"}).
			BorderForeground(theme.Mauve).
			PaddingLeft(1)
	} else if block.Type == pipeline.BlockTypeAI {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "│"}).
			BorderForeground(theme.Blue).
			PaddingLeft(1)
	} else if block.ExitCode != 0 {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "│"}).
			BorderForeground(theme.Red).
			PaddingLeft(1)
	} else {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "│"}).
			BorderForeground(theme.Green).
			PaddingLeft(1)
	}

	var blockContent strings.Builder

	if block.Type == pipeline.BlockTypeAI {
		queryStyle := lipgloss.NewStyle().Foreground(theme.Blue).Bold(true)
		blockContent.WriteString(queryStyle.Render("? " + block.Command))
	} else {
		promptStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)
		cmdStyle := lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
		blockContent.WriteString(promptStyle.Render("❯ ") + cmdStyle.Render(block.Command))

		metaStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		meta := metaStyle.Render(fmt.Sprintf("  %s", block.Timestamp.Format("15:04:05")))

		if block.ExitCode != 0 {
			exitStyle := lipgloss.NewStyle().Foreground(theme.Red).Bold(true)
			meta += " " + exitStyle.Render(fmt.Sprintf("✗ %d", block.ExitCode))
		}

		if block.Duration > 0 {
			meta += metaStyle.Render(fmt.Sprintf(" (%dms)", block.Duration.Milliseconds()))
		}

		blockContent.WriteString(meta)
	}

	if block.Folded {
		foldStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		blockContent.WriteString(foldStyle.Render(" ▸"))
	}

	blockContent.WriteString("\n")

	if !block.Folded && block.Output != "" {
		outputStyle := lipgloss.NewStyle().Foreground(theme.Text)
		if block.Type == pipeline.BlockTypeAI {
			outputStyle = outputStyle.Foreground(theme.Subtext0)
		}

		lines := strings.Split(block.Output, "\n")
		maxLines := 50
		if len(lines) > maxLines {
			for _, line := range lines[:maxLines] {
				blockContent.WriteString(outputStyle.Render(line) + "\n")
			}
			moreStyle := lipgloss.NewStyle().Foreground(theme.Yellow).Italic(true)
			blockContent.WriteString(moreStyle.Render(fmt.Sprintf("... +%d lines (press z to fold)", len(lines)-maxLines)) + "\n")
		} else {
			blockContent.WriteString(outputStyle.Render(block.Output) + "\n")
		}
	}

	if block.AISuggestion != "" {
		fixStyle := lipgloss.NewStyle().
			Background(theme.Surface0).
			Foreground(theme.Yellow).
			Padding(0, 1)

		blockContent.WriteString("\n")
		blockContent.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render("💡 AI: "))
		blockContent.WriteString(fixStyle.Render(block.AISuggestion))

		actionsStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
		blockContent.WriteString("\n   " + actionsStyle.Render("[r]un") + " " + actionsStyle.Render("[c]opy") + " " + actionsStyle.Render("[d]ismiss"))
	}

	suggestions := m.State().GetSuggestionsForBlock(block.ID)
	if len(suggestions) > 0 && block.AISuggestion == "" {
		sug := suggestions[0]
		sugStyle := lipgloss.NewStyle().
			Background(theme.Surface0).
			Foreground(theme.Yellow).
			Padding(0, 1)

		blockContent.WriteString("\n")
		blockContent.WriteString(lipgloss.NewStyle().Foreground(theme.Yellow).Render("💡 "))
		blockContent.WriteString(sugStyle.Render(sug.Explanation))
	}

	return borderStyle.Width(width).Render(blockContent.String())
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
		hint = hintStyle.Render("  [i]nsert [?]AI [j/k]nav [z]fold")
	} else {
		hint = hintStyle.Render("  [Enter]run [Esc]normal [?]ask AI")
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

func (m Model) renderStarshipBar(width int) string {
	statusStyle := lipgloss.NewStyle().
		Background(theme.Surface0).
		Foreground(theme.Text).
		Width(width).
		Padding(0, 1)

	return statusStyle.Render(m.StarshipLine())
}

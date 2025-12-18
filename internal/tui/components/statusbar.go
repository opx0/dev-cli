package components

import (
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	Width int
	help  help.Model
}

func NewStatusBar() StatusBar {
	h := help.New()
	h.ShowAll = false
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(theme.Overlay0)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(theme.Surface2)
	return StatusBar{
		help: h,
	}
}

func (s StatusBar) SetWidth(w int) StatusBar {
	s.Width = w
	return s
}

func (s StatusBar) Render(keys help.KeyMap, focusLabel string) string {
	helpView := s.help.View(keys)

	focusStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	bracketStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	focusIndicator := bracketStyle.Render(" │ [") + focusStyle.Render(focusLabel) + bracketStyle.Render("]")

	content := helpView + focusIndicator
	return theme.StatusBar.Width(s.Width).MaxWidth(s.Width).Render(content)
}

func (s StatusBar) RenderWithInfo(keys help.KeyMap, focusLabel string, info string) string {
	helpView := s.help.View(keys)

	focusStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)

	bracketStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0)

	focusIndicator := bracketStyle.Render(" │ [") + focusStyle.Render(focusLabel) + bracketStyle.Render("]")

	infoStr := ""
	if info != "" {
		infoStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		infoStr = bracketStyle.Render(" │ ") + infoStyle.Render(info)
	}

	content := helpView + focusIndicator + infoStr
	return theme.StatusBar.Width(s.Width).MaxWidth(s.Width).Render(content)
}

func (s StatusBar) RenderSimple(text string) string {
	return theme.StatusBar.Width(s.Width).Render(text)
}

package components

import (
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/bubbles/help"
)

type StatusBar struct {
	Width int
	help  help.Model
}

func NewStatusBar() StatusBar {
	h := help.New()
	h.ShowAll = false
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

	focusIndicator := theme.StatusDesc.Render(" │ [" + focusLabel + "]")

	content := helpView + focusIndicator
	return theme.StatusBar.Width(s.Width).Render(content)
}

func (s StatusBar) RenderSimple(text string) string {
	return theme.StatusBar.Width(s.Width).Render(text)
}

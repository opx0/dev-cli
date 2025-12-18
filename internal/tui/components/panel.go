package components

import (
	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

type FocusState int

const (
	FocusBlurred FocusState = iota
	FocusFocused
	FocusInsert
)

type Panel struct {
	Title  string
	Width  int
	Height int
	Focus  FocusState
}

func NewPanel(title string) Panel {
	return Panel{
		Title: title,
		Focus: FocusBlurred,
	}
}

func (p Panel) SetSize(w, h int) Panel {
	p.Width = w
	p.Height = h
	return p
}

func (p Panel) SetFocus(f FocusState) Panel {
	p.Focus = f
	return p
}

func (p Panel) Style() lipgloss.Style {
	switch p.Focus {
	case FocusInsert:
		return theme.InsertModePanel
	case FocusFocused:
		return theme.FocusedPanel
	default:
		return theme.Panel
	}
}

func (p Panel) Render(content string) string {
	style := p.Style().Width(p.Width).Height(p.Height)

	header := ""
	if p.Title != "" {
		header = theme.Header.Render(p.Title) + "\n"
	}

	return style.Render(header + content)
}

func (p Panel) RenderWithHeader(header, content string) string {
	style := p.Style().Width(p.Width).Height(p.Height)
	return style.Render(header + "\n" + content)
}

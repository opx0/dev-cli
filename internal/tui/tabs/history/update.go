package history

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	Details  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("j/k", "nav"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", ""),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "focus"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "details"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn", "page down"),
		),
	}
}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Tab):
			if m.focus == FocusSidebar {
				m.focus = FocusMain
			} else {
				m.focus = FocusSidebar
			}

		case key.Matches(msg, keys.Up):
			if m.focus == FocusSidebar {

				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
				m.updateDetailsContent()
			} else {
				m.viewport.ScrollUp(1)
			}

		case key.Matches(msg, keys.Down):
			if m.focus == FocusSidebar {
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
				m.updateDetailsContent()
			} else {
				m.viewport.ScrollDown(1)
			}

		case key.Matches(msg, keys.PageUp):
			if m.focus == FocusSidebar {
				m.list.Paginator.PrevPage()
				m.updateDetailsContent()
			} else {
				m.viewport.HalfPageUp()
			}

		case key.Matches(msg, keys.PageDown):
			if m.focus == FocusSidebar {
				m.list.Paginator.NextPage()
				m.updateDetailsContent()
			} else {
				m.viewport.HalfPageDown()
			}

		case key.Matches(msg, keys.Details):
			if m.focus == FocusSidebar {
				m.focus = FocusMain
			}
		}
	}

	if m.focus == FocusMain {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

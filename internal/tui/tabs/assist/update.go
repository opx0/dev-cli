package assist

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Insert   key.Binding
	Escape   key.Binding
	Tab      key.Binding
	Enter    key.Binding
	ToggleAI key.Binding
	Up       key.Binding
	Down     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Insert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "chat"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "focus"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "send"),
		),
		ToggleAI: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("Ctrl+t", "toggle AI"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("j/k", "scroll"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", ""),
		),
	}
}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.insertMode {

			switch {
			case key.Matches(msg, keys.Escape):
				m = m.SetInsertMode(false)
				return m, nil

			case key.Matches(msg, keys.Enter):

				return m, nil
			}

			var cmd tea.Cmd
			ti := m.input
			ti, cmd = ti.Update(msg)
			m.input = ti
			cmds = append(cmds, cmd)

		} else {

			switch {
			case key.Matches(msg, keys.Insert):
				m = m.SetInsertMode(true)

			case key.Matches(msg, keys.Tab):
				if m.focus == FocusSidebar {
					m.focus = FocusMain
				} else {
					m.focus = FocusSidebar
				}

			case key.Matches(msg, keys.ToggleAI):
				m = m.ToggleAIMode()

			case key.Matches(msg, keys.Up):
				if m.focus == FocusMain {
					m.viewport.LineUp(1)
				}

			case key.Matches(msg, keys.Down):
				if m.focus == FocusMain {
					m.viewport.LineDown(1)
				}
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

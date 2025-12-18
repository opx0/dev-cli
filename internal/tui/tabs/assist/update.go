package assist

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Insert   key.Binding
	Escape   key.Binding
	Enter    key.Binding
	ToggleAI key.Binding
	Up       key.Binding
	Down     key.Binding
	Clear    key.Binding
	Copy     key.Binding
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
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("Ctrl+l", "clear"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
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
				value := m.input.Value()
				if value != "" {
					m = m.AddUserMessage(value)
					m.input.SetValue("")
				}
				return m, nil

			case key.Matches(msg, keys.ToggleAI):
				m = m.ToggleAIMode()
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

			case key.Matches(msg, keys.ToggleAI):
				m = m.ToggleAIMode()

			case key.Matches(msg, keys.Up):
				m.viewport.ScrollUp(1)

			case key.Matches(msg, keys.Down):
				m.viewport.ScrollDown(1)

			case key.Matches(msg, keys.Clear):
				m = m.ClearMessages()

			case msg.String() == "g":
				m.viewport.GotoTop()

			case msg.String() == "G":
				m.viewport.GotoBottom()
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Insert    key.Binding
	Escape    key.Binding
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Actions   key.Binding
	Fold      key.Binding
	Clear     key.Binding
	PrevBlock key.Binding
	NextBlock key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Insert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("j/k", "nav blocks"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", ""),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "submit"),
		),
		Actions: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "actions"),
		),
		Fold: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "fold"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("Ctrl+l", "clear"),
		),
		PrevBlock: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[[", "prev block"),
		),
		NextBlock: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]]", "next block"),
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
				cmd := m.input.Value()
				if cmd != "" {
					m = m.AddOutputBlock(cmd)
					m.input.SetValue("")
				}
				return m, nil
			}

			var cmd tea.Cmd
			ti := m.input
			ti, cmd = ti.Update(msg)
			m.input = ti
			cmds = append(cmds, cmd)

		} else {
			switch {
			case key.Matches(msg, keys.Insert), key.Matches(msg, keys.Enter):
				m = m.SetInsertMode(true)

			case key.Matches(msg, keys.Up):
				if len(m.outputBlocks) > 0 {
					if m.selectedBlock > 0 {
						m.selectedBlock--
					} else if m.selectedBlock == -1 {
						m.selectedBlock = len(m.outputBlocks) - 1
					}
				} else {
					m.viewport.ScrollUp(1)
				}

			case key.Matches(msg, keys.Down):
				if len(m.outputBlocks) > 0 {
					if m.selectedBlock < len(m.outputBlocks)-1 {
						m.selectedBlock++
					}
				} else {
					m.viewport.ScrollDown(1)
				}

			case key.Matches(msg, keys.PrevBlock):
				if m.selectedBlock > 0 {
					m.selectedBlock--
				}

			case key.Matches(msg, keys.NextBlock):
				if m.selectedBlock < len(m.outputBlocks)-1 {
					m.selectedBlock++
				}

			case key.Matches(msg, keys.Fold):
				if m.selectedBlock >= 0 && m.selectedBlock < len(m.outputBlocks) {
					m = m.ToggleFoldBlock(m.selectedBlock)
				}

			case key.Matches(msg, keys.Actions):
				m.showingActions = !m.showingActions

			case key.Matches(msg, keys.Clear):
				m.outputBlocks = []OutputBlock{}
				m.selectedBlock = -1
				m = m.ClearViewport()

			case msg.String() == "g":
				if len(m.outputBlocks) > 0 {
					m.selectedBlock = 0
				}

			case msg.String() == "G":
				if len(m.outputBlocks) > 0 {
					m.selectedBlock = len(m.outputBlocks) - 1
				}
			}

			if m.showingActions {
				switch msg.String() {
				case "r":
					if m.selectedBlock >= 0 && m.selectedBlock < len(m.outputBlocks) {
						cmd := m.outputBlocks[m.selectedBlock].Command
						m = m.AddOutputBlock(cmd)
					}
					m.showingActions = false
				case "c":
					m.outputBlocks = []OutputBlock{}
					m.selectedBlock = -1
					m.showingActions = false
				case "esc", "q":
					m.showingActions = false
				}
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

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
			// Handle insert mode keys
			switch {
			case key.Matches(msg, keys.Escape):
				m = m.SetInsertMode(false)
				return m, nil

			case key.Matches(msg, keys.Enter):
				// Submit command
				cmd := m.input.Value()
				if cmd != "" {
					m = m.AddOutputBlock(cmd)
					m.input.SetValue("")
				}
				return m, nil
			}

			// Pass to text input
			var cmd tea.Cmd
			ti := m.input
			ti, cmd = ti.Update(msg)
			m.input = ti
			cmds = append(cmds, cmd)

		} else {
			// Handle normal mode keys
			switch {
			case key.Matches(msg, keys.Insert), key.Matches(msg, keys.Enter):
				m = m.SetInsertMode(true)

			case key.Matches(msg, keys.Up):
				// Navigate blocks up
				if len(m.outputBlocks) > 0 {
					if m.selectedBlock > 0 {
						m.selectedBlock--
					} else if m.selectedBlock == -1 {
						m.selectedBlock = len(m.outputBlocks) - 1
					}
				} else {
					m.viewport.LineUp(1)
				}

			case key.Matches(msg, keys.Down):
				// Navigate blocks down
				if len(m.outputBlocks) > 0 {
					if m.selectedBlock < len(m.outputBlocks)-1 {
						m.selectedBlock++
					}
				} else {
					m.viewport.LineDown(1)
				}

			case key.Matches(msg, keys.PrevBlock):
				// Jump to previous block (double tap support)
				if m.selectedBlock > 0 {
					m.selectedBlock--
				}

			case key.Matches(msg, keys.NextBlock):
				// Jump to next block (double tap support)
				if m.selectedBlock < len(m.outputBlocks)-1 {
					m.selectedBlock++
				}

			case key.Matches(msg, keys.Fold):
				// Toggle fold on selected block
				if m.selectedBlock >= 0 && m.selectedBlock < len(m.outputBlocks) {
					m = m.ToggleFoldBlock(m.selectedBlock)
				}

			case key.Matches(msg, keys.Actions):
				// Toggle action menu
				m.showingActions = !m.showingActions

			case key.Matches(msg, keys.Clear):
				// Clear output blocks
				m.outputBlocks = []OutputBlock{}
				m.selectedBlock = -1
				m = m.ClearViewport()

			case msg.String() == "g":
				// Go to first block
				if len(m.outputBlocks) > 0 {
					m.selectedBlock = 0
				}

			case msg.String() == "G":
				// Go to last block
				if len(m.outputBlocks) > 0 {
					m.selectedBlock = len(m.outputBlocks) - 1
				}
			}

			// Handle action menu navigation
			if m.showingActions {
				switch msg.String() {
				case "r":
					// Retry command
					if m.selectedBlock >= 0 && m.selectedBlock < len(m.outputBlocks) {
						cmd := m.outputBlocks[m.selectedBlock].Command
						m = m.AddOutputBlock(cmd)
					}
					m.showingActions = false
				case "c":
					// Clear
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

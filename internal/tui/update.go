package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "tab":
			m.focus = (m.focus + 1) % 4

		case "shift+tab":
			m.focus = (m.focus + 3) % 4

		case "1":
			m.focus = FocusContainers
		case "2":
			m.focus = FocusObservability
		case "3":
			m.focus = FocusKeybinds
		case "4":
			m.focus = FocusTerminal

		case "enter":
			if m.focus == FocusTerminal && m.input.Value() != "" {
				m.cmdHistory = append(m.cmdHistory, "> "+m.input.Value())
				m.input.SetValue("")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 30

	case dockerHealthMsg:
		m.dockerHealth = msg.health
		if msg.health.Available {
			m.state = StateMain
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.focus == FocusTerminal {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

package monitor

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Tab         key.Binding
	ScrollLeft  key.Binding
	ScrollRight key.Binding
	ToggleWrap  key.Binding
	ResetScroll key.Binding
	TriggerRCA  key.Binding
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
		ScrollLeft: key.NewBinding(
			key.WithKeys("H", "shift+left"),
			key.WithHelp("H/L", "scroll"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("L", "shift+right"),
			key.WithHelp("", ""),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("Ctrl+w", "wrap"),
		),
		ResetScroll: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "reset"),
		),
		TriggerRCA: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "RCA"),
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

				if len(m.dockerHealth.Containers) > 0 {
					m.containerCursor--
					if m.containerCursor < 0 {
						m.containerCursor = len(m.dockerHealth.Containers) - 1
					}
				}
			} else {
				m.viewport.LineUp(1)
			}

		case key.Matches(msg, keys.Down):
			if m.focus == FocusSidebar {

				if len(m.dockerHealth.Containers) > 0 {
					m.containerCursor = (m.containerCursor + 1) % len(m.dockerHealth.Containers)
				}
			} else {
				m.viewport.LineDown(1)
			}

		case key.Matches(msg, keys.ScrollLeft):
			if !m.wrapMode {
				m.horizontalOffset -= 10
				if m.horizontalOffset < 0 {
					m.horizontalOffset = 0
				}
			}

		case key.Matches(msg, keys.ScrollRight):
			if !m.wrapMode {
				maxScroll := m.maxLineWidth - (m.width - 34)
				if maxScroll < 0 {
					maxScroll = 0
				}
				m.horizontalOffset += 10
				if m.horizontalOffset > maxScroll {
					m.horizontalOffset = maxScroll
				}
			}

		case key.Matches(msg, keys.ToggleWrap):
			m = m.ToggleWrapMode()

		case key.Matches(msg, keys.ResetScroll):
			m.horizontalOffset = 0

		case key.Matches(msg, keys.TriggerRCA):

		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

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
	Follow      key.Binding
	LogLevel    key.Binding
	Actions     key.Binding
	Top         key.Binding
	Bottom      key.Binding
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
			key.WithHelp("Tab", "panel"),
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
			key.WithHelp("?", "AI RCA"),
		),
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow"),
		),
		LogLevel: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("L", "filter"),
		),
		Actions: key.NewBinding(
			key.WithKeys("a", "enter"),
			key.WithHelp("a", "actions"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g/G", "top/bottom"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("", ""),
		),
	}
}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showingActions {
			return m.handleActionMenu(msg)
		}

		switch {
		case key.Matches(msg, keys.Tab):
			switch m.focus {
			case FocusList:
				m.focus = FocusLogs
			case FocusLogs:
				m.focus = FocusStats
			case FocusStats:
				m.focus = FocusList
			}

		case key.Matches(msg, keys.Up):
			switch m.focus {
			case FocusList:
				if len(m.dockerHealth.Containers) > 0 {
					m.containerCursor--
					if m.containerCursor < 0 {
						m.containerCursor = len(m.dockerHealth.Containers) - 1
					}
				}
			case FocusLogs:
				m.viewport.ScrollUp(1)
				m.followMode = false
			}

		case key.Matches(msg, keys.Down):
			switch m.focus {
			case FocusList:
				if len(m.dockerHealth.Containers) > 0 {
					m.containerCursor = (m.containerCursor + 1) % len(m.dockerHealth.Containers)
				}
			case FocusLogs:
				m.viewport.ScrollDown(1)
			}

		case key.Matches(msg, keys.ScrollLeft):
			if !m.wrapMode && m.focus == FocusLogs {
				m.horizontalOffset -= 10
				if m.horizontalOffset < 0 {
					m.horizontalOffset = 0
				}
			}

		case key.Matches(msg, keys.ScrollRight):
			if !m.wrapMode && m.focus == FocusLogs {
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

		case key.Matches(msg, keys.Follow):
			m = m.ToggleFollowMode()

		case key.Matches(msg, keys.LogLevel):
			m = m.CycleLogLevelFilter()

		case key.Matches(msg, keys.Actions):
			if m.focus == FocusList && len(m.dockerHealth.Containers) > 0 {
				m.showingActions = true
				m.actionMenuIndex = 0
			}

		case key.Matches(msg, keys.Top):
			if m.focus == FocusLogs {
				m.viewport.GotoTop()
			} else if m.focus == FocusList && len(m.dockerHealth.Containers) > 0 {
				m.containerCursor = 0
			}

		case key.Matches(msg, keys.Bottom):
			if m.focus == FocusLogs {
				m.viewport.GotoBottom()
			} else if m.focus == FocusList && len(m.dockerHealth.Containers) > 0 {
				m.containerCursor = len(m.dockerHealth.Containers) - 1
			}

		case key.Matches(msg, keys.TriggerRCA):
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) handleActionMenu(msg tea.KeyMsg) (Model, tea.Cmd) {
	actionCount := 7 // Number of menu items

	switch msg.String() {
	case "j", "down":
		m.actionMenuIndex = (m.actionMenuIndex + 1) % actionCount
	case "k", "up":
		m.actionMenuIndex--
		if m.actionMenuIndex < 0 {
			m.actionMenuIndex = actionCount - 1
		}
	case "esc", "q":
		m.showingActions = false
	case "enter":
		m.showingActions = false
	case "s":
		m.showingActions = false
	case "l":
		m.followMode = true
		m.showingActions = false
	case "r":
		m.showingActions = false
	case "x":
		m.showingActions = false
	}

	return m, nil
}

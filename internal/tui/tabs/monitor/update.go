package monitor

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	Follow   key.Binding
	LogLevel key.Binding
	Record   key.Binding
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Top      key.Binding
	Bottom   key.Binding
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
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow"),
		),
		LogLevel: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("L", "filter"),
		),
		Record: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "record"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
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

// Message types
type ContainerActionMsg struct {
	Action      string
	ContainerID string
	Success     bool
	Error       error
}

type RefreshContainersMsg struct{}
type RefreshImagesMsg struct{}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Tab):
			// Cycle focus: Services → Logs → Images → Stats → Services
			switch m.focus {
			case FocusServices:
				m.focus = FocusLogs
			case FocusLogs:
				m.focus = FocusImages
			case FocusImages:
				m.focus = FocusStats
			case FocusStats:
				m.focus = FocusServices
			}

		case key.Matches(msg, keys.Up):
			switch m.focus {
			case FocusServices:
				var cmd tea.Cmd
				m.servicesList, cmd = m.servicesList.Update(msg)
				cmds = append(cmds, cmd)
			case FocusImages:
				var cmd tea.Cmd
				m.imagesList, cmd = m.imagesList.Update(msg)
				cmds = append(cmds, cmd)
			case FocusLogs:
				m.viewport.ScrollUp(1)
				m.followMode = false
			}

		case key.Matches(msg, keys.Down):
			switch m.focus {
			case FocusServices:
				var cmd tea.Cmd
				m.servicesList, cmd = m.servicesList.Update(msg)
				cmds = append(cmds, cmd)
			case FocusImages:
				var cmd tea.Cmd
				m.imagesList, cmd = m.imagesList.Update(msg)
				cmds = append(cmds, cmd)
			case FocusLogs:
				m.viewport.ScrollDown(1)
			}

		case key.Matches(msg, keys.Follow):
			m = m.ToggleFollowMode()

		case key.Matches(msg, keys.LogLevel):
			m = m.CycleLogLevelFilter()

		case key.Matches(msg, keys.Record):
			m = m.ToggleRecording()

		case key.Matches(msg, keys.Top):
			switch m.focus {
			case FocusLogs:
				m.viewport.GotoTop()
			case FocusServices:
				m.servicesList.Select(0)
			case FocusImages:
				m.imagesList.Select(0)
			}

		case key.Matches(msg, keys.Bottom):
			switch m.focus {
			case FocusLogs:
				m.viewport.GotoBottom()
			case FocusServices:
				if len(m.services) > 0 {
					m.servicesList.Select(len(m.services) - 1)
				}
			case FocusImages:
				if len(m.images) > 0 {
					m.imagesList.Select(len(m.images) - 1)
				}
			}

		case key.Matches(msg, keys.Start):
			// Will be handled by parent to call Docker client
			if m.focus == FocusServices {
				if svc := m.SelectedService(); svc != nil {
					return m, func() tea.Msg {
						return ContainerActionMsg{
							Action:      "start",
							ContainerID: svc.ID,
						}
					}
				}
			}

		case key.Matches(msg, keys.Stop):
			if m.focus == FocusServices {
				if svc := m.SelectedService(); svc != nil {
					return m, func() tea.Msg {
						return ContainerActionMsg{
							Action:      "stop",
							ContainerID: svc.ID,
						}
					}
				}
			}

		case key.Matches(msg, keys.Restart):
			if m.focus == FocusServices {
				if svc := m.SelectedService(); svc != nil {
					return m, func() tea.Msg {
						return ContainerActionMsg{
							Action:      "restart",
							ContainerID: svc.ID,
						}
					}
				}
			}
		}
	}

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

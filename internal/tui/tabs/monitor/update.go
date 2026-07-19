package monitor

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	LogLevel key.Binding
	Top      key.Binding
	Bottom   key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k")),
		Down:     key.NewBinding(key.WithKeys("down", "j")),
		LogLevel: key.NewBinding(key.WithKeys("l")),
		Top:      key.NewBinding(key.WithKeys("g")),
		Bottom:   key.NewBinding(key.WithKeys("G")),
	}
}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, keys.Up):
		if m.selected > 0 {
			m.selected--
		}
	case key.Matches(keyMsg, keys.Down):
		if m.selected+1 < len(m.services) {
			m.selected++
		}
	case key.Matches(keyMsg, keys.Top):
		m.selected = 0
	case key.Matches(keyMsg, keys.Bottom):
		if len(m.services) > 0 {
			m.selected = len(m.services) - 1
		}
	case key.Matches(keyMsg, keys.LogLevel):
		switch m.logLevelFilter {
		case "":
			m.logLevelFilter = "ERROR"
		case "ERROR":
			m.logLevelFilter = "WARN"
		case "WARN":
			m.logLevelFilter = "INFO"
		default:
			m.logLevelFilter = ""
		}
	}
	return m, nil
}

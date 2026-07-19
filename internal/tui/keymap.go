package tui

import "github.com/charmbracelet/bubbles/key"

type MonitorKeyMap struct {
	Up, Down, Filter, Switch, Quit key.Binding
}

func (k MonitorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Filter, k.Switch, k.Quit}
}

func (k MonitorKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

var MonitorKeys = MonitorKeyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("j/k", "select")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Filter: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "filter logs")),
	Switch: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "history")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type HistoryKeyMap struct {
	Up, Down, Details, Switch, Quit key.Binding
}

func (k HistoryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Details, k.Switch, k.Quit}
}

func (k HistoryKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

var HistoryKeys = HistoryKeyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("j/k", "select")),
	Down:    key.NewBinding(key.WithKeys("down", "j")),
	Details: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
	Switch:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "containers")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

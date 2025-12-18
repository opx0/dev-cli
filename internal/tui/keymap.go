package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type GlobalKeyMap struct {
	Quit   key.Binding
	Tab    key.Binding
	Insert key.Binding
	Escape key.Binding
	Up     key.Binding
	Down   key.Binding
	Tab1   key.Binding
	Tab2   key.Binding
	Tab3   key.Binding
}

func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab1, k.Tab2, k.Tab3, k.Tab, k.Quit}
}

func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Tab},
		{k.Insert, k.Escape, k.Quit},
	}
}

var GlobalKeys = GlobalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "focus"),
	),
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
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "agent"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "containers"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "history"),
	),
}

// AgentKeyMap for Agent tab
type AgentKeyMap struct {
	GlobalKeyMap
	Fold     key.Binding
	Clear    key.Binding
	ToggleAI key.Binding
	RunFix   key.Binding
}

func (k AgentKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Insert, k.Fold, k.ToggleAI, k.Clear, k.Quit}
}

func (k AgentKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Insert, k.Fold, k.Clear},
		{k.ToggleAI, k.RunFix},
		{k.Up, k.Down, k.Quit},
	}
}

var AgentKeys = AgentKeyMap{
	GlobalKeyMap: GlobalKeys,
	Fold: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "fold"),
	),
	Clear: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("Ctrl+l", "clear"),
	),
	ToggleAI: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("Ctrl+t", "AI mode"),
	),
	RunFix: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "run fix"),
	),
}

// MonitorKeyMap for Containers tab
type MonitorKeyMap struct {
	GlobalKeyMap
	Follow     key.Binding
	LogLevel   key.Binding
	Actions    key.Binding
	ToggleWrap key.Binding
}

func (k MonitorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Follow, k.LogLevel, k.Actions, k.Quit}
}

func (k MonitorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Tab},
		{k.Follow, k.LogLevel, k.ToggleWrap},
		{k.Actions, k.Quit},
	}
}

var MonitorKeys = MonitorKeyMap{
	GlobalKeyMap: GlobalKeys,
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
	ToggleWrap: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("Ctrl+w", "wrap"),
	),
}

// HistoryKeyMap for History tab
type HistoryKeyMap struct {
	GlobalKeyMap
	Details key.Binding
}

func (k HistoryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Details, k.Tab, k.Quit}
}

func (k HistoryKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Details},
		{k.Tab, k.Quit},
	}
}

var HistoryKeys = HistoryKeyMap{
	GlobalKeyMap: GlobalKeys,
	Details: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "details"),
	),
}

func NewHelp() help.Model {
	h := help.New()
	h.ShowAll = false
	return h
}

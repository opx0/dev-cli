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
	Tab4   key.Binding
}

func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab1, k.Tab2, k.Tab3, k.Tab4, k.Tab, k.Quit}
}

func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
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
		key.WithHelp("1", "dashboard"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "monitor"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "assist"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "history"),
	),
}

type DashboardKeyMap struct {
	GlobalKeyMap
	ScrollUp   key.Binding
	ScrollDown key.Binding
}

func (k DashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Insert, k.ScrollUp, k.ScrollDown, k.Tab, k.Quit}
}

func (k DashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.Insert, k.ScrollUp, k.ScrollDown},
		{k.Tab, k.Quit},
	}
}

var DashboardKeys = DashboardKeyMap{
	GlobalKeyMap: GlobalKeys,
	ScrollUp: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("j/k", "scroll"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("", ""),
	),
}

type MonitorKeyMap struct {
	GlobalKeyMap
	ScrollLeft  key.Binding
	ScrollRight key.Binding
	ToggleWrap  key.Binding
	ResetScroll key.Binding
	TriggerRCA  key.Binding
}

func (k MonitorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.ScrollLeft, k.ToggleWrap, k.TriggerRCA, k.Quit}
}

func (k MonitorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.Up, k.Down, k.ScrollLeft, k.ScrollRight},
		{k.ToggleWrap, k.ResetScroll, k.TriggerRCA},
		{k.Tab, k.Quit},
	}
}

var MonitorKeys = MonitorKeyMap{
	GlobalKeyMap: GlobalKeys,
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

type AssistKeyMap struct {
	GlobalKeyMap
	ToggleAI key.Binding
}

func (k AssistKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Insert, k.ToggleAI, k.Tab, k.Quit}
}

func (k AssistKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.Insert, k.ToggleAI},
		{k.Tab, k.Quit},
	}
}

var AssistKeys = AssistKeyMap{
	GlobalKeyMap: GlobalKeys,
	ToggleAI: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("Ctrl+t", "toggle AI"),
	),
}

type HistoryKeyMap struct {
	GlobalKeyMap
	Details key.Binding
}

func (k HistoryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Details, k.Tab, k.Quit}
}

func (k HistoryKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
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

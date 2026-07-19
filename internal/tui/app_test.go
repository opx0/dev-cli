package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelNavigationAndQuit(t *testing.T) {
	model := InitialModel()
	model.state = StateMain
	if model.activeTab != TabContainers {
		t.Fatal("containers should be the initial tab")
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	if model.activeTab != TabHistory {
		t.Fatal("tab should select history")
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = next.(Model)
	if !model.quitting || cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestModelResizeAndView(t *testing.T) {
	model := InitialModel()
	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = next.(Model)
	model.state = StateMain
	if model.width != 100 || model.height != 30 || model.View() == "" {
		t.Fatal("resize or view failed")
	}
}

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	model := InitialModel()

	if model.state != StateLoading {
		t.Errorf("expected StateLoading, got %v", model.state)
	}
	if model.mode != ModeNormal {
		t.Errorf("expected ModeNormal, got %v", model.mode)
	}
	if model.activeTab != TabAgent {
		t.Errorf("expected TabAgent as initial tab, got %v", model.activeTab)
	}
	if model.quitting {
		t.Error("quitting should be false initially")
	}
}

func TestModel_TabSwitching(t *testing.T) {
	model := InitialModel()
	model.state = StateMain

	tabMsg := tea.KeyMsg{Type: tea.KeyTab}

	newModel, _ := model.Update(tabMsg)
	m := newModel.(Model)

	if m.activeTab != TabContainers {
		t.Errorf("expected TabContainers after first tab, got %v", m.activeTab)
	}

	newModel, _ = m.Update(tabMsg)
	m = newModel.(Model)

	if m.activeTab != TabHistory {
		t.Errorf("expected TabHistory after second tab, got %v", m.activeTab)
	}

	newModel, _ = m.Update(tabMsg)
	m = newModel.(Model)

	if m.activeTab != TabAgent {
		t.Errorf("expected TabAgent after wrap, got %v", m.activeTab)
	}
}

func TestModel_QuitOnCtrlC(t *testing.T) {
	model := InitialModel()
	model.state = StateMain

	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.Update(ctrlC)
	m := newModel.(Model)

	if !m.quitting {
		t.Error("quitting should be true after Ctrl+C")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestModel_WindowResize(t *testing.T) {
	model := InitialModel()

	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := model.Update(resizeMsg)
	m := newModel.(Model)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestModel_ModeFromTab(t *testing.T) {
	tests := []struct {
		tab      Tab
		expected AppMode
	}{
		{TabAgent, ModeNormal}, // Agent starts in normal mode
		{TabContainers, ModeNormal},
		{TabHistory, ModeNormal},
	}

	for _, tt := range tests {
		model := InitialModel()
		model.activeTab = tt.tab

		got := model.getModeFromTab()
		if got != tt.expected {
			t.Errorf("getModeFromTab() for tab %v = %v, want %v", tt.tab, got, tt.expected)
		}
	}
}

func TestModel_ViewRendering(t *testing.T) {
	model := InitialModel()
	model.width = 80
	model.height = 24

	model.state = StateLoading
	loadingView := model.View()
	if loadingView == "" {
		t.Error("loading view should not be empty")
	}

	model.state = StateMain
	mainView := model.View()
	if mainView == "" {
		t.Error("main view should not be empty")
	}
}

func TestModel_ViewLoading(t *testing.T) {
	model := InitialModel()
	model.width = 80
	model.height = 24
	model.state = StateLoading

	view := model.viewLoading()

	if view == "" {
		t.Error("viewLoading should return content")
	}
}

func TestModel_ViewMain(t *testing.T) {
	model := InitialModel()
	model.width = 80
	model.height = 24
	model.state = StateMain

	view := model.viewMain()

	if view == "" {
		t.Error("viewMain should return content")
	}
}

func TestModel_GetFocusLabel(t *testing.T) {
	model := InitialModel()
	model.state = StateMain

	tests := []struct {
		tab      Tab
		contains string
	}{
		{TabAgent, "Agent"},
		{TabContainers, ""},
		{TabHistory, "History"},
	}

	for _, tt := range tests {
		model.activeTab = tt.tab
		label := model.getFocusLabel()

		if tt.contains != "" && !strings.Contains(label, tt.contains) {
			t.Errorf("getFocusLabel() for tab %v should contain %q, got %q",
				tt.tab, tt.contains, label)
		}
	}
}

func TestModel_Init(t *testing.T) {
	model := InitialModel()

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModel_ShiftTabReverse(t *testing.T) {
	model := InitialModel()
	model.state = StateMain
	model.activeTab = TabAgent

	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := model.Update(shiftTabMsg)
	m := newModel.(Model)

	if m.activeTab != TabHistory {
		t.Errorf("expected TabHistory after Shift+Tab from first tab, got %v", m.activeTab)
	}
}

func TestModel_EscapeKey(t *testing.T) {
	model := InitialModel()
	model.state = StateMain
	model.mode = ModeInsert

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.Update(escMsg)
	m := newModel.(Model)

	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after Escape, got %v", m.mode)
	}
}

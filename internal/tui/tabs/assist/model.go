package assist

import (
	"dev-cli/internal/llm"
	"dev-cli/internal/tui/components"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type FocusPanel int

const (
	FocusSidebar FocusPanel = iota
	FocusMain
)

type Model struct {
	width  int
	height int
	focus  FocusPanel

	sidebar  components.Panel
	chat     components.Panel
	viewport viewport.Model
	input    textinput.Model

	chatHistory []string
	aiClient    *llm.HybridClient
	aiMode      string

	insertMode bool
}

func New(aiClient *llm.HybridClient, aiMode string) Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything..."
	ti.CharLimit = 512
	ti.Width = 60

	vp := viewport.New(0, 0)

	return Model{
		sidebar:  components.NewPanel(" ◈ AI Model"),
		chat:     components.NewPanel(" ◇ Chat"),
		viewport: vp,
		input:    ti,
		aiClient: aiClient,
		aiMode:   aiMode,
		focus:    FocusSidebar,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	sidebarWidth := 20
	chatWidth := w - sidebarWidth - 4
	panelHeight := h - 6

	if chatWidth < 40 {
		chatWidth = 40
	}
	if panelHeight < 10 {
		panelHeight = 10
	}

	m.sidebar = m.sidebar.SetSize(sidebarWidth, panelHeight)
	m.chat = m.chat.SetSize(chatWidth, panelHeight)
	m.viewport.Width = chatWidth - 4
	m.viewport.Height = panelHeight - 6
	m.input.Width = chatWidth - 10

	return m
}

func (m Model) SetFocus(f FocusPanel) Model {
	m.focus = f
	return m
}

func (m Model) SetInsertMode(insert bool) Model {
	m.insertMode = insert
	if insert {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
	return m
}

func (m Model) ToggleAIMode() Model {
	if m.aiMode == "local" {
		m.aiMode = "cloud"
	} else {
		m.aiMode = "local"
	}
	return m
}

func (m Model) AddMessage(msg string) Model {
	m.chatHistory = append(m.chatHistory, msg)
	m.viewport.SetContent(m.formatChatHistory())
	m.viewport.GotoBottom()
	return m
}

func (m Model) formatChatHistory() string {
	result := ""
	for _, msg := range m.chatHistory {
		result += msg + "\n"
	}
	return result
}

func (m Model) Focus() FocusPanel {
	return m.focus
}

func (m Model) InsertMode() bool {
	return m.insertMode
}

func (m Model) AIMode() string {
	return m.aiMode
}

func (m Model) ChatHistory() []string {
	return m.chatHistory
}

func (m Model) Input() textinput.Model {
	return m.input
}

func (m Model) SetInput(ti textinput.Model) Model {
	m.input = ti
	return m
}

func (m Model) Viewport() viewport.Model {
	return m.viewport
}

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) ClearInput() Model {
	m.input.SetValue("")
	return m
}

func (m Model) InputValue() string {
	return m.input.Value()
}

func (m Model) Width() int {
	return m.width
}

func (m Model) Height() int {
	return m.height
}

func (m Model) MessageCount() int {
	return len(m.chatHistory)
}

func (m Model) SetChatHistory(history []string) Model {
	m.chatHistory = history
	m.viewport.SetContent(m.formatChatHistory())
	return m
}

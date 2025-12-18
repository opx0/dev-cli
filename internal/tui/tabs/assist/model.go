package assist

import (
	"dev-cli/internal/llm"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

// ChatMessage represents a single message in the chat
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

type Model struct {
	width  int
	height int

	viewport viewport.Model
	input    textinput.Model

	// Chat state
	messages    []ChatMessage
	chatHistory []string // Legacy compatibility
	aiClient    *llm.HybridClient
	aiMode      string // "local" or "cloud"

	// UI state
	insertMode bool
	isLoading  bool

	// Context awareness
	recentCommands int
	containerCount int
	errorCount     int
}

func New(aiClient *llm.HybridClient, aiMode string) Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything..."
	ti.CharLimit = 1024
	ti.Width = 60

	vp := viewport.New(0, 0)

	return Model{
		viewport: vp,
		input:    ti,
		aiClient: aiClient,
		aiMode:   aiMode,
		messages: []ChatMessage{},
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	contentHeight := h - 8 // Header + input area
	if contentHeight < 10 {
		contentHeight = 10
	}

	m.viewport.Width = w - 4
	m.viewport.Height = contentHeight
	m.input.Width = w - 12

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

// AddMessage adds a new message to the chat
func (m Model) AddMessage(role, content string) Model {
	msg := ChatMessage{
		Role:    role,
		Content: content,
	}
	m.messages = append(m.messages, msg)
	m.chatHistory = append(m.chatHistory, content) // Legacy compatibility
	return m
}

func (m Model) AddUserMessage(content string) Model {
	return m.AddMessage("user", content)
}

func (m Model) AddAssistantMessage(content string) Model {
	return m.AddMessage("assistant", content)
}

func (m Model) Messages() []ChatMessage {
	return m.messages
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
	return len(m.messages)
}

func (m Model) SetChatHistory(history []string) Model {
	m.chatHistory = history
	return m
}

func (m Model) IsLoading() bool {
	return m.isLoading
}

func (m Model) SetLoading(loading bool) Model {
	m.isLoading = loading
	return m
}

// Context awareness setters
func (m Model) SetRecentCommands(count int) Model {
	m.recentCommands = count
	return m
}

func (m Model) SetContainerCount(count int) Model {
	m.containerCount = count
	return m
}

func (m Model) SetErrorCount(count int) Model {
	m.errorCount = count
	return m
}

func (m Model) RecentCommands() int {
	return m.recentCommands
}

func (m Model) ContainerCount() int {
	return m.containerCount
}

func (m Model) ErrorCount() int {
	return m.errorCount
}

// ClearMessages clears all messages
func (m Model) ClearMessages() Model {
	m.messages = []ChatMessage{}
	m.chatHistory = []string{}
	return m
}

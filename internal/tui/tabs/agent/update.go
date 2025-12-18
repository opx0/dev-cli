package agent

import (
	"dev-cli/internal/executor"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/plugins/command"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Insert   key.Binding
	Escape   key.Binding
	Enter    key.Binding
	Up       key.Binding
	Down     key.Binding
	Fold     key.Binding
	Clear    key.Binding
	ToggleAI key.Binding
	RunFix   key.Binding
	Dismiss  key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Insert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "run"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("j/k", "nav"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", ""),
		),
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
		Dismiss: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "dismiss"),
		),
	}
}

// CommandExecutedMsg is sent when a command finishes executing
type CommandExecutedMsg struct {
	BlockID string
}

// AIResponseMsg is sent when AI responds
type AIResponseMsg struct {
	BlockID  string
	Response string
	Error    error
}

func (m Model) Update(msg tea.Msg, keys KeyMap) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CommandExecutedMsg:
		// Command finished - state already updated by plugin
		m.isExecuting = false
		blocks := m.Blocks()
		if len(blocks) > 0 {
			m.selectedBlock = len(blocks) - 1
		}
		return m, nil

	case AIResponseMsg:
		m.isExecuting = false
		return m, nil

	case tea.KeyMsg:
		if m.insertMode {
			// Insert mode key handling
			switch {
			case key.Matches(msg, keys.Escape):
				m = m.SetInsertMode(false)
				return m, nil

			case key.Matches(msg, keys.Enter):
				input := m.input.Value()
				if input == "" {
					return m, nil
				}

				m.input.SetValue("")

				// Check if AI query
				if executor.IsAIQuery(input) {
					queryType, query := executor.ParseAIQuery(input)
					return m.handleAIQuery(queryType, query)
				}

				// Execute as shell command through pipeline
				m.isExecuting = true
				return m, executeCommandPipeline(m.cmdPlugin, input)

			case key.Matches(msg, keys.ToggleAI):
				// Toggle could be stored in state
				return m, nil
			}

			// Pass to text input
			var cmd tea.Cmd
			ti := m.input
			ti, cmd = ti.Update(msg)
			m.input = ti
			cmds = append(cmds, cmd)

		} else {
			// Normal mode key handling
			switch {
			case key.Matches(msg, keys.Insert):
				m = m.SetInsertMode(true)

			case key.Matches(msg, keys.Up):
				blocks := m.Blocks()
				if len(blocks) > 0 {
					if m.selectedBlock > 0 {
						m.selectedBlock--
					} else if m.selectedBlock == -1 {
						m.selectedBlock = len(blocks) - 1
					}
				}

			case key.Matches(msg, keys.Down):
				blocks := m.Blocks()
				if len(blocks) > 0 && m.selectedBlock < len(blocks)-1 {
					m.selectedBlock++
				}

			case key.Matches(msg, keys.Fold):
				m = m.ToggleFoldBlock(m.selectedBlock)

			case key.Matches(msg, keys.Clear):
				m = m.ClearBlocks()

			case key.Matches(msg, keys.RunFix):
				// Run the AI suggested fix
				blocks := m.Blocks()
				if m.selectedBlock >= 0 && m.selectedBlock < len(blocks) {
					block := blocks[m.selectedBlock]
					if block.AISuggestion != "" {
						m.isExecuting = true
						return m, executeCommandPipeline(m.cmdPlugin, block.AISuggestion)
					}
					// Check state suggestions
					suggestions := m.State().GetSuggestionsForBlock(block.ID)
					if len(suggestions) > 0 && suggestions[0].Command != "" {
						m.isExecuting = true
						return m, executeCommandPipeline(m.cmdPlugin, suggestions[0].Command)
					}
				}

			case key.Matches(msg, keys.Dismiss):
				// Dismiss AI fix
				blocks := m.Blocks()
				if m.selectedBlock >= 0 && m.selectedBlock < len(blocks) {
					block := blocks[m.selectedBlock]
					m.State().UpdateBlock(block.ID, func(b *pipeline.Block) {
						b.AISuggestion = ""
					})
				}

			case msg.String() == "g":
				blocks := m.Blocks()
				if len(blocks) > 0 {
					m.selectedBlock = 0
				}

			case msg.String() == "G":
				blocks := m.Blocks()
				if len(blocks) > 0 {
					m.selectedBlock = len(blocks) - 1
				}

			case msg.String() == "?":
				// Quick enter insert mode with ?
				m = m.SetInsertMode(true)
				m.input.SetValue("?")
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) handleAIQuery(queryType, query string) (Model, tea.Cmd) {
	m.isExecuting = true

	switch queryType {
	case "fix":
		// Get last command block for context
		blocks := m.Blocks()
		for i := len(blocks) - 1; i >= 0; i-- {
			if blocks[i].Type == pipeline.BlockTypeCommand && blocks[i].ExitCode != 0 {
				return m, requestAIFix(m.cmdPlugin, blocks[i])
			}
		}
		m = m.ExecuteAIQuery("No previous error to fix")
		m.isExecuting = false
		return m, nil

	case "explain":
		blocks := m.Blocks()
		for i := len(blocks) - 1; i >= 0; i-- {
			if blocks[i].Type == pipeline.BlockTypeCommand {
				return m, requestAIExplain(m.cmdPlugin, blocks[i])
			}
		}
		m = m.ExecuteAIQuery("No previous command to explain")
		m.isExecuting = false
		return m, nil

	case "question":
		return m, requestAIQuestion(m.cmdPlugin, query)

	default:
		return m, requestAIQuestion(m.cmdPlugin, query)
	}
}

// Commands for async operations
func executeCommandPipeline(cmdPlugin *command.Plugin, cmd string) tea.Cmd {
	return func() tea.Msg {
		if cmdPlugin != nil {
			block := cmdPlugin.Execute(cmd)
			return CommandExecutedMsg{BlockID: block.ID}
		}
		return CommandExecutedMsg{BlockID: ""}
	}
}

func requestAIQuestion(cmdPlugin *command.Plugin, query string) tea.Cmd {
	return func() tea.Msg {
		if cmdPlugin != nil {
			block := cmdPlugin.ExecuteAI(query)
			return AIResponseMsg{BlockID: block.ID}
		}
		return AIResponseMsg{BlockID: ""}
	}
}

func requestAIFix(cmdPlugin *command.Plugin, block pipeline.Block) tea.Cmd {
	return func() tea.Msg {
		if cmdPlugin != nil {
			b := cmdPlugin.ExecuteAI("Fix: " + block.Command + "\nError: " + block.Output)
			return AIResponseMsg{BlockID: b.ID}
		}
		return AIResponseMsg{BlockID: ""}
	}
}

func requestAIExplain(cmdPlugin *command.Plugin, block pipeline.Block) tea.Cmd {
	return func() tea.Msg {
		if cmdPlugin != nil {
			b := cmdPlugin.ExecuteAI("Explain: " + block.Command + "\nOutput: " + block.Output)
			return AIResponseMsg{BlockID: b.ID}
		}
		return AIResponseMsg{BlockID: ""}
	}
}

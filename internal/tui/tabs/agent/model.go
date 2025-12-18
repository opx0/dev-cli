package agent

import (
	"dev-cli/internal/infra"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/plugins/ai"
	"dev-cli/internal/plugins/command"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type Model struct {
	width  int
	height int

	viewport viewport.Model
	input    textinput.Model

	pipeline  *pipeline.Pipeline
	cmdPlugin *command.Plugin
	aiPlugin  *ai.Plugin

	insertMode    bool
	isExecuting   bool
	selectedBlock int
}

func New(pipe *pipeline.Pipeline) Model {
	ti := textinput.New()
	ti.Placeholder = "command or ?question..."
	ti.CharLimit = 1024
	ti.Width = 60

	vp := viewport.New(0, 0)

	var cmdPlugin *command.Plugin
	var aiPlugin *ai.Plugin

	if p := pipe.GetPlugin("command"); p != nil {
		cmdPlugin = p.(*command.Plugin)
	}
	if p := pipe.GetPlugin("ai"); p != nil {
		aiPlugin = p.(*ai.Plugin)
	}

	return Model{
		viewport:      vp,
		input:         ti,
		pipeline:      pipe,
		cmdPlugin:     cmdPlugin,
		aiPlugin:      aiPlugin,
		selectedBlock: -1,
	}
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	contentHeight := h - 8
	if contentHeight < 5 {
		contentHeight = 5
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

func (m Model) InsertMode() bool {
	return m.insertMode
}

func (m Model) IsExecuting() bool {
	return m.isExecuting
}

func (m Model) SetExecuting(exec bool) Model {
	m.isExecuting = exec
	return m
}

func (m Model) ExecuteCommand(cmd string) Model {
	if m.cmdPlugin != nil {
		m.cmdPlugin.Execute(cmd)
		m.selectedBlock = len(m.State().Blocks) - 1
	}
	return m
}

func (m Model) ExecuteAIQuery(query string) Model {
	if m.cmdPlugin != nil {
		block := m.cmdPlugin.ExecuteAI(query)

		if m.aiPlugin != nil {
			response, _ := m.aiPlugin.AnswerQuery(query, block.ID)
			m.State().UpdateBlock(block.ID, func(b *pipeline.Block) {
				b.Output = response
			})
		}
		m.selectedBlock = len(m.State().Blocks) - 1
	}
	return m
}

func (m Model) State() *pipeline.StateStore {
	return m.pipeline.State()
}

func (m Model) Blocks() []pipeline.Block {
	return m.State().GetBlocks()
}

func (m Model) SelectedBlock() int {
	return m.selectedBlock
}

func (m Model) SetSelectedBlock(idx int) Model {
	blocks := m.Blocks()
	if idx >= -1 && idx < len(blocks) {
		m.selectedBlock = idx
	}
	return m
}

func (m Model) ToggleFoldBlock(idx int) Model {
	blocks := m.Blocks()
	if idx >= 0 && idx < len(blocks) {
		m.State().UpdateBlock(blocks[idx].ID, func(b *pipeline.Block) {
			b.Folded = !b.Folded
		})
	}
	return m
}

func (m Model) ClearBlocks() Model {
	m.State().ClearBlocks()
	m.selectedBlock = -1
	return m
}

func (m Model) Input() textinput.Model {
	return m.input
}

func (m Model) SetInput(ti textinput.Model) Model {
	m.input = ti
	return m
}

func (m Model) InputValue() string {
	return m.input.Value()
}

func (m Model) ClearInput() Model {
	m.input.SetValue("")
	return m
}

func (m Model) Width() int {
	return m.width
}

func (m Model) Height() int {
	return m.height
}

func (m Model) Cwd() string {
	return m.State().Cwd
}

func (m Model) DockerHealth() infra.DockerHealth {
	return m.State().DockerHealth
}

func (m Model) GPUStats() infra.GPUStats {
	return m.State().GPUStats
}

func (m Model) GitStatus() infra.GitStatus {
	return m.State().GitStatus
}

func (m Model) StarshipLine() string {
	return m.State().StarshipLine
}

func (m Model) AIMode() string {
	return "local"
}

func (m Model) Viewport() viewport.Model {
	return m.viewport
}

func (m Model) SetViewport(vp viewport.Model) Model {
	m.viewport = vp
	return m
}

func (m Model) BlockCount() int {
	return len(m.Blocks())
}

func (m Model) GetSuggestions() []pipeline.Suggestion {
	if m.selectedBlock < 0 || m.selectedBlock >= len(m.Blocks()) {
		return nil
	}
	block := m.Blocks()[m.selectedBlock]
	return m.State().GetSuggestionsForBlock(block.ID)
}

func (m Model) HasLastError() bool {
	return m.State().LastError != nil
}

func (m Model) SetCwd(cwd string) Model {
	m.State().SetCwd(cwd)
	return m
}

func (m Model) SetDockerHealth(h infra.DockerHealth) Model {
	m.State().SetDockerHealth(h)
	return m
}

func (m Model) SetGPUStats(s infra.GPUStats) Model {
	m.State().SetGPUStats(s)
	return m
}

func (m Model) SetGitStatus(g infra.GitStatus) Model {
	m.State().SetGitStatus(g)
	return m
}

func (m Model) SetStarshipLine(line string) Model {
	m.State().SetStarshipLine(line)
	return m
}

func (m Model) Publish(event pipeline.Event) {
	m.pipeline.Publish(event)
}

func (m Model) Subscribe(eventType pipeline.EventType, handler pipeline.EventHandler) {
	m.pipeline.Subscribe(eventType, handler)
}

func (m Model) GetContext() map[string]interface{} {
	return m.State().GetContext()
}

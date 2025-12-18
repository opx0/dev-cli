package command

import (
	"context"
	"time"

	"dev-cli/internal/executor"
	"dev-cli/internal/pipeline"

	"github.com/google/uuid"
)

// Plugin handles command execution and publishes to pipeline
type Plugin struct {
	bus   *pipeline.EventBus
	state *pipeline.StateStore
}

// New creates a new command plugin
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "command"
}

func (p *Plugin) Init(bus *pipeline.EventBus, state *pipeline.StateStore) error {
	p.bus = bus
	p.state = state
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	// Command plugin is reactive - Execute is called explicitly
	return nil
}

func (p *Plugin) Stop() error {
	return nil
}

// Execute runs a command and publishes events/updates state
func (p *Plugin) Execute(command string) pipeline.Block {
	blockID := uuid.New().String()

	// Publish start event
	p.bus.Publish(pipeline.Event{
		Type:      pipeline.EventCommandStart,
		Timestamp: time.Now(),
		Source:    p.Name(),
		BlockID:   blockID,
		Data: map[string]string{
			"command": command,
		},
	})

	// Execute using PTY for full interactive shell (aliases work!)
	result := executor.ExecutePTY(command)

	// Create block
	block := pipeline.Block{
		ID:         blockID,
		Type:       pipeline.BlockTypeCommand,
		Timestamp:  result.Timestamp,
		Command:    result.Command,
		Output:     result.Output,
		ExitCode:   result.ExitCode,
		Duration:   result.Duration,
		WorkingDir: p.state.Cwd,
		GitBranch:  p.state.GitStatus.Branch,
	}

	// Add to state
	p.state.AddBlock(block)

	// Publish completion event
	eventType := pipeline.EventCommandComplete
	if result.ExitCode != 0 {
		eventType = pipeline.EventCommandError
		block.Type = pipeline.BlockTypeError
	}

	p.bus.Publish(pipeline.Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    p.Name(),
		BlockID:   blockID,
		Data:      block,
	})

	return block
}

// ExecuteAI handles AI query execution
func (p *Plugin) ExecuteAI(query string) pipeline.Block {
	blockID := uuid.New().String()

	block := pipeline.Block{
		ID:        blockID,
		Type:      pipeline.BlockTypeAI,
		Timestamp: time.Now(),
		Command:   query,
		// Output will be filled by AI plugin
	}

	p.state.AddBlock(block)

	// Publish event for AI plugin to handle
	p.bus.Publish(pipeline.Event{
		Type:      pipeline.EventAISuggestion,
		Timestamp: time.Now(),
		Source:    p.Name(),
		BlockID:   blockID,
		Data: map[string]string{
			"query": query,
		},
	})

	return block
}

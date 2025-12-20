package command

import (
	"context"
	"time"

	"dev-cli/internal/executor"
	"dev-cli/internal/pipeline"

	"github.com/google/uuid"
)

type Plugin struct {
	bus   *pipeline.EventBus
	state *pipeline.StateStore
}

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
	return nil
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Execute(command string) pipeline.Block {
	blockID := uuid.New().String()

	p.bus.Publish(pipeline.Event{
		Type:      pipeline.EventCommandStart,
		Timestamp: time.Now(),
		Source:    p.Name(),
		BlockID:   blockID,
		Data: map[string]string{
			"command": command,
		},
	})

	result := executor.ExecutePTY(command)

	block := pipeline.Block{
		ID:         blockID,
		Type:       pipeline.BlockTypeCommand,
		Timestamp:  result.Timestamp,
		Command:    result.Command,
		Output:     result.Output,
		ExitCode:   result.ExitCode,
		Duration:   result.Duration,
		WorkingDir: p.state.Cwd,
	}

	p.state.AddBlock(block)

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

func (p *Plugin) ExecuteAI(query string) pipeline.Block {
	blockID := uuid.New().String()

	block := pipeline.Block{
		ID:        blockID,
		Type:      pipeline.BlockTypeAI,
		Timestamp: time.Now(),
		Command:   query,
	}

	p.state.AddBlock(block)

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

package pipeline

import (
	"context"
	"sync"
)

// Plugin interface that all plugins must implement
type Plugin interface {
	// Name returns the plugin identifier
	Name() string

	// Init is called with the event bus and state store
	Init(bus *EventBus, state *StateStore) error

	// Start begins the plugin's work (may run in background)
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin
	Stop() error
}

// Pipeline manages the event bus, state store, and plugins
type Pipeline struct {
	mu      sync.RWMutex
	bus     *EventBus
	state   *StateStore
	plugins map[string]Plugin
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPipeline creates a new pipeline
func NewPipeline() *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		bus:     NewEventBus(),
		state:   NewStateStore(),
		plugins: make(map[string]Plugin),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Bus returns the event bus
func (p *Pipeline) Bus() *EventBus {
	return p.bus
}

// State returns the state store
func (p *Pipeline) State() *StateStore {
	return p.state
}

// Register adds a plugin to the pipeline
func (p *Pipeline) Register(plugin Plugin) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := plugin.Init(p.bus, p.state); err != nil {
		return err
	}

	p.plugins[plugin.Name()] = plugin
	return nil
}

// Start begins all registered plugins
func (p *Pipeline) Start() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.plugins {
		if err := plugin.Start(p.ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all plugins
func (p *Pipeline) Stop() error {
	p.cancel()

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.plugins {
		if err := plugin.Stop(); err != nil {
			// Log error but continue stopping others
		}
	}
	return nil
}

// GetPlugin returns a plugin by name
func (p *Pipeline) GetPlugin(name string) Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.plugins[name]
}

// Publish is a convenience method to publish events
func (p *Pipeline) Publish(event Event) {
	p.bus.Publish(event)
}

// Subscribe is a convenience method to subscribe to events
func (p *Pipeline) Subscribe(eventType EventType, handler EventHandler) {
	p.bus.Subscribe(eventType, handler)
}

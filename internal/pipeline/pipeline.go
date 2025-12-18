package pipeline

import (
	"context"
	"sync"
)

type Plugin interface {
	Name() string

	Init(bus *EventBus, state *StateStore) error

	Start(ctx context.Context) error

	Stop() error
}

type Pipeline struct {
	mu      sync.RWMutex
	bus     *EventBus
	state   *StateStore
	plugins map[string]Plugin
	ctx     context.Context
	cancel  context.CancelFunc
}

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

func (p *Pipeline) Bus() *EventBus {
	return p.bus
}

func (p *Pipeline) State() *StateStore {
	return p.state
}

func (p *Pipeline) Register(plugin Plugin) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := plugin.Init(p.bus, p.state); err != nil {
		return err
	}

	p.plugins[plugin.Name()] = plugin
	return nil
}

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

func (p *Pipeline) Stop() error {
	p.cancel()

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.plugins {
		if err := plugin.Stop(); err != nil {
		}
	}
	return nil
}

func (p *Pipeline) GetPlugin(name string) Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.plugins[name]
}

func (p *Pipeline) Publish(event Event) {
	p.bus.Publish(event)
}

func (p *Pipeline) Subscribe(eventType EventType, handler EventHandler) {
	p.bus.Subscribe(eventType, handler)
}

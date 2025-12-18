package pipeline

import (
	"sync"
	"time"
)

// EventType defines the type of event in the pipeline
type EventType string

const (
	// Command events
	EventCommandStart    EventType = "command.start"
	EventCommandOutput   EventType = "command.output"
	EventCommandComplete EventType = "command.complete"
	EventCommandError    EventType = "command.error"

	// Container events
	EventContainerLog    EventType = "container.log"
	EventContainerStatus EventType = "container.status"
	EventContainerAlert  EventType = "container.alert"

	// Git events
	EventGitStatus  EventType = "git.status"
	EventGitChanged EventType = "git.changed"

	// AI events
	EventAISuggestion EventType = "ai.suggestion"
	EventAIAnalysis   EventType = "ai.analysis"

	// System events
	EventSystemAlert EventType = "system.alert"
	EventSystemStats EventType = "system.stats"
)

// Event represents a message in the pipeline
type Event struct {
	Type      EventType
	Timestamp time.Time
	Source    string      // plugin/component name
	Data      interface{} // typed payload
	BlockID   string      // optional: links to a Block
}

// EventHandler is called when an event is received
type EventHandler func(Event)

// EventBus is the central message broker
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]EventHandler
	history     []Event // recent events for context
	maxHistory  int
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		history:     make([]Event, 0),
		maxHistory:  100,
	}
}

// Subscribe adds a handler for a specific event type
func (e *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers[eventType] = append(e.subscribers[eventType], handler)
}

// SubscribeAll adds a handler that receives all events
func (e *EventBus) SubscribeAll(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Use empty string as "all events" key
	e.subscribers["*"] = append(e.subscribers["*"], handler)
}

// Publish sends an event to all subscribers
func (e *EventBus) Publish(event Event) {
	e.mu.Lock()
	// Add to history
	e.history = append(e.history, event)
	if len(e.history) > e.maxHistory {
		e.history = e.history[1:]
	}

	// Get subscribers
	handlers := make([]EventHandler, 0)
	handlers = append(handlers, e.subscribers[event.Type]...)
	handlers = append(handlers, e.subscribers["*"]...)
	e.mu.Unlock()

	// Call handlers (outside lock) - synchronously to avoid race conditions
	// with bubbletea's update loop
	for _, handler := range handlers {
		handler(event)
	}
}

// RecentEvents returns the last N events
func (e *EventBus) RecentEvents(n int) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if n > len(e.history) {
		n = len(e.history)
	}
	return e.history[len(e.history)-n:]
}

// RecentByType returns recent events of a specific type
func (e *EventBus) RecentByType(eventType EventType, n int) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []Event
	for i := len(e.history) - 1; i >= 0 && len(result) < n; i-- {
		if e.history[i].Type == eventType {
			result = append(result, e.history[i])
		}
	}
	return result
}

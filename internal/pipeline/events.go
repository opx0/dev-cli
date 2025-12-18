package pipeline

import (
	"sync"
	"time"
)

type EventType string

const (
	EventCommandStart    EventType = "command.start"
	EventCommandOutput   EventType = "command.output"
	EventCommandComplete EventType = "command.complete"
	EventCommandError    EventType = "command.error"

	EventContainerLog    EventType = "container.log"
	EventContainerStatus EventType = "container.status"
	EventContainerAlert  EventType = "container.alert"

	EventGitStatus  EventType = "git.status"
	EventGitChanged EventType = "git.changed"

	EventAISuggestion EventType = "ai.suggestion"
	EventAIAnalysis   EventType = "ai.analysis"

	EventSystemAlert EventType = "system.alert"
	EventSystemStats EventType = "system.stats"
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Source    string
	Data      interface{}
	BlockID   string
}

type EventHandler func(Event)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]EventHandler
	history     []Event
	maxHistory  int
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		history:     make([]Event, 0),
		maxHistory:  100,
	}
}

func (e *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers[eventType] = append(e.subscribers[eventType], handler)
}

func (e *EventBus) SubscribeAll(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers["*"] = append(e.subscribers["*"], handler)
}

func (e *EventBus) Publish(event Event) {
	e.mu.Lock()
	e.history = append(e.history, event)
	if len(e.history) > e.maxHistory {
		e.history = e.history[1:]
	}

	handlers := make([]EventHandler, 0)
	handlers = append(handlers, e.subscribers[event.Type]...)
	handlers = append(handlers, e.subscribers["*"]...)
	e.mu.Unlock()

	for _, handler := range handlers {
		handler(event)
	}
}

func (e *EventBus) RecentEvents(n int) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if n > len(e.history) {
		n = len(e.history)
	}
	return e.history[len(e.history)-n:]
}

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

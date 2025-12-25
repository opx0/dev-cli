package core

import (
	"sync"
	"time"

	"dev-cli/internal/infra"
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

	EventWorkflowStart      EventType = "workflow.start"
	EventWorkflowStep       EventType = "workflow.step"
	EventWorkflowCheckpoint EventType = "workflow.checkpoint"
	EventWorkflowComplete   EventType = "workflow.complete"
	EventWorkflowRollback   EventType = "workflow.rollback"

	EventRCAStart     EventType = "rca.start"
	EventRCANodeFound EventType = "rca.node_found"
	EventRCAComplete  EventType = "rca.complete"
	EventRCACacheHit  EventType = "rca.cache_hit"

	EventRemediationPending    EventType = "remediation.pending"
	EventRemediationApproved   EventType = "remediation.approved"
	EventRemediationExecuted   EventType = "remediation.executed"
	EventRemediationRolledBack EventType = "remediation.rollback"
	EventRemediationSkipped    EventType = "remediation.skipped"
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

type BlockType string

const (
	BlockTypeCommand    BlockType = "command"
	BlockTypeAI         BlockType = "ai"
	BlockTypeOutput     BlockType = "output"
	BlockTypeError      BlockType = "error"
	BlockTypeSuggestion BlockType = "suggestion"
)

type Block struct {
	ID        string
	Type      BlockType
	Timestamp time.Time
	Command   string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Folded    bool

	AISuggestion string
	AIAnalyzed   bool

	WorkingDir string
}

type Suggestion struct {
	ForBlockID  string
	Type        string
	Title       string
	Command     string
	Explanation string
	Confidence  float64
}

type StateStore struct {
	mu sync.RWMutex

	Blocks      []Block
	blockIndex  map[string]int
	SelectedIdx int
	MaxBlocks   int

	DockerHealth infra.DockerHealth
	GPUStats     infra.GPUStats
	StarshipLine string

	Suggestions   []Suggestion
	LastError     *Block
	ErrorPatterns map[string]string

	Cwd       string
	Shell     string
	IsLoading bool
}

func NewStateStore() *StateStore {
	return &StateStore{
		Blocks:        make([]Block, 0),
		blockIndex:    make(map[string]int),
		SelectedIdx:   -1,
		MaxBlocks:     100,
		Suggestions:   make([]Suggestion, 0),
		ErrorPatterns: make(map[string]string),
	}
}

func (s *StateStore) AddBlock(block Block) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Blocks) >= s.MaxBlocks {
		oldest := s.Blocks[0]
		delete(s.blockIndex, oldest.ID)
		s.Blocks = s.Blocks[1:]
		s.rebuildIndex()
	}

	s.Blocks = append(s.Blocks, block)
	s.blockIndex[block.ID] = len(s.Blocks) - 1
	s.SelectedIdx = len(s.Blocks) - 1

	if block.ExitCode != 0 {
		s.LastError = &block
	}
}

func (s *StateStore) GetBlock(id string) *Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx, ok := s.blockIndex[id]; ok && idx < len(s.Blocks) {
		return &s.Blocks[idx]
	}
	return nil
}

func (s *StateStore) GetRecentBlocks(n int) []Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n > len(s.Blocks) {
		n = len(s.Blocks)
	}
	result := make([]Block, n)
	copy(result, s.Blocks[len(s.Blocks)-n:])
	return result
}

func (s *StateStore) GetBlocks() []Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Block, len(s.Blocks))
	copy(result, s.Blocks)
	return result
}

func (s *StateStore) UpdateBlock(id string, fn func(*Block)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, ok := s.blockIndex[id]; ok && idx < len(s.Blocks) {
		fn(&s.Blocks[idx])
	}
}

func (s *StateStore) AddSuggestion(suggestion Suggestion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Suggestions = append(s.Suggestions, suggestion)
	if len(s.Suggestions) > 10 {
		s.Suggestions = s.Suggestions[1:]
	}
}

func (s *StateStore) GetSuggestionsForBlock(blockID string) []Suggestion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Suggestion
	for _, sug := range s.Suggestions {
		if sug.ForBlockID == blockID {
			result = append(result, sug)
		}
	}
	return result
}

func (s *StateStore) ClearBlocks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Blocks = make([]Block, 0)
	s.blockIndex = make(map[string]int)
	s.SelectedIdx = -1
}

func (s *StateStore) rebuildIndex() {
	s.blockIndex = make(map[string]int, len(s.Blocks))
	for i, block := range s.Blocks {
		s.blockIndex[block.ID] = i
	}
}

func (s *StateStore) SetDockerHealth(h infra.DockerHealth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DockerHealth = h
}

func (s *StateStore) SetGPUStats(g infra.GPUStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GPUStats = g
}

func (s *StateStore) SetStarshipLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StarshipLine = line
}

func (s *StateStore) SetCwd(cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cwd = cwd
}

func (s *StateStore) GetContext() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cwd":             s.Cwd,
		"container_count": len(s.DockerHealth.Containers),
		"has_last_error":  s.LastError != nil,
		"recent_commands": len(s.Blocks),
	}
}

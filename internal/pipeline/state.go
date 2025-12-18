package pipeline

import (
	"sync"
	"time"

	"dev-cli/internal/infra"
)

// BlockType represents the type of content block (like Warp's blocks)
type BlockType string

const (
	BlockTypeCommand    BlockType = "command"
	BlockTypeAI         BlockType = "ai"
	BlockTypeOutput     BlockType = "output"
	BlockTypeError      BlockType = "error"
	BlockTypeSuggestion BlockType = "suggestion"
)

// Block represents a unit of content (inspired by Warp's block architecture)
// Each command and its output is a "block" that can be referenced, shared, analyzed
type Block struct {
	ID        string
	Type      BlockType
	Timestamp time.Time
	Command   string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Folded    bool

	// AI-related
	AISuggestion string
	AIAnalyzed   bool

	// Context
	WorkingDir string
	GitBranch  string
}

// Suggestion represents an AI suggestion for a block
type Suggestion struct {
	ForBlockID  string
	Type        string // "fix", "explain", "related"
	Title       string
	Command     string
	Explanation string
	Confidence  float64
}

// StateStore holds shared state accessible to all components
type StateStore struct {
	mu sync.RWMutex

	// Blocks (warp-style command/output blocks)
	Blocks      []Block
	SelectedIdx int
	MaxBlocks   int

	// System status
	DockerHealth infra.DockerHealth
	GPUStats     infra.GPUStats
	GitStatus    infra.GitStatus
	StarshipLine string

	// AI context
	Suggestions   []Suggestion
	LastError     *Block
	ErrorPatterns map[string]string // pattern -> fix

	// Working context
	Cwd       string
	Shell     string
	IsLoading bool
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		Blocks:        make([]Block, 0),
		SelectedIdx:   -1,
		MaxBlocks:     100,
		Suggestions:   make([]Suggestion, 0),
		ErrorPatterns: make(map[string]string),
	}
}

// AddBlock adds a new block to the state
func (s *StateStore) AddBlock(block Block) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Blocks = append(s.Blocks, block)
	if len(s.Blocks) > s.MaxBlocks {
		s.Blocks = s.Blocks[1:]
	}
	s.SelectedIdx = len(s.Blocks) - 1

	// Track last error for AI context
	if block.ExitCode != 0 {
		s.LastError = &block
	}
}

// GetBlock returns a block by ID
func (s *StateStore) GetBlock(id string) *Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.Blocks {
		if s.Blocks[i].ID == id {
			return &s.Blocks[i]
		}
	}
	return nil
}

// GetRecentBlocks returns the last N blocks
func (s *StateStore) GetRecentBlocks(n int) []Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n > len(s.Blocks) {
		n = len(s.Blocks)
	}
	// Return a copy to avoid race conditions
	result := make([]Block, n)
	copy(result, s.Blocks[len(s.Blocks)-n:])
	return result
}

// GetBlocks returns a copy of all blocks (thread-safe)
func (s *StateStore) GetBlocks() []Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Block, len(s.Blocks))
	copy(result, s.Blocks)
	return result
}

// UpdateBlock updates a block by ID
func (s *StateStore) UpdateBlock(id string, fn func(*Block)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Blocks {
		if s.Blocks[i].ID == id {
			fn(&s.Blocks[i])
			return
		}
	}
}

// AddSuggestion adds an AI suggestion
func (s *StateStore) AddSuggestion(suggestion Suggestion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Suggestions = append(s.Suggestions, suggestion)
	// Keep max 10 suggestions
	if len(s.Suggestions) > 10 {
		s.Suggestions = s.Suggestions[1:]
	}
}

// GetSuggestionsForBlock returns suggestions for a specific block
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

// ClearBlocks removes all blocks
func (s *StateStore) ClearBlocks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Blocks = make([]Block, 0)
	s.SelectedIdx = -1
}

// SetDockerHealth updates docker status
func (s *StateStore) SetDockerHealth(h infra.DockerHealth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DockerHealth = h
}

// SetGitStatus updates git status
func (s *StateStore) SetGitStatus(g infra.GitStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GitStatus = g
}

// SetGPUStats updates GPU stats
func (s *StateStore) SetGPUStats(g infra.GPUStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GPUStats = g
}

// SetStarshipLine updates the starship prompt line
func (s *StateStore) SetStarshipLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StarshipLine = line
}

// SetCwd updates current working directory
func (s *StateStore) SetCwd(cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cwd = cwd
}

// GetContext returns a context summary for AI
func (s *StateStore) GetContext() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cwd":             s.Cwd,
		"git_branch":      s.GitStatus.Branch,
		"git_changes":     s.GitStatus.Modified + s.GitStatus.Added + s.GitStatus.Deleted,
		"container_count": len(s.DockerHealth.Containers),
		"has_last_error":  s.LastError != nil,
		"recent_commands": len(s.Blocks),
	}
}

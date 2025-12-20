package pipeline

import (
	"sync"
	"time"

	"dev-cli/internal/infra"
)

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
	blockIndex  map[string]int // O(1) lookup by ID -> slice index
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

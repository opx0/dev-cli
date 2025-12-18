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
	GitBranch  string
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
	SelectedIdx int
	MaxBlocks   int

	DockerHealth infra.DockerHealth
	GPUStats     infra.GPUStats
	GitStatus    infra.GitStatus
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
		SelectedIdx:   -1,
		MaxBlocks:     100,
		Suggestions:   make([]Suggestion, 0),
		ErrorPatterns: make(map[string]string),
	}
}

func (s *StateStore) AddBlock(block Block) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Blocks = append(s.Blocks, block)
	if len(s.Blocks) > s.MaxBlocks {
		s.Blocks = s.Blocks[1:]
	}
	s.SelectedIdx = len(s.Blocks) - 1

	if block.ExitCode != 0 {
		s.LastError = &block
	}
}

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

	for i := range s.Blocks {
		if s.Blocks[i].ID == id {
			fn(&s.Blocks[i])
			return
		}
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
	s.SelectedIdx = -1
}

func (s *StateStore) SetDockerHealth(h infra.DockerHealth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DockerHealth = h
}

func (s *StateStore) SetGitStatus(g infra.GitStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GitStatus = g
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
		"git_branch":      s.GitStatus.Branch,
		"git_changes":     s.GitStatus.Modified + s.GitStatus.Added + s.GitStatus.Deleted,
		"container_count": len(s.DockerHealth.Containers),
		"has_last_error":  s.LastError != nil,
		"recent_commands": len(s.Blocks),
	}
}

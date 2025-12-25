package pipeline

import (
	"sync"
	"testing"
	"time"

	"dev-cli/internal/infra"
)

func TestStateStore_AddBlock(t *testing.T) {
	store := NewStateStore()

	block := Block{
		ID:        "block-1",
		Type:      BlockTypeCommand,
		Timestamp: time.Now(),
		Command:   "ls -la",
	}
	store.AddBlock(block)

	if len(store.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(store.Blocks))
	}
	if store.SelectedIdx != 0 {
		t.Errorf("expected SelectedIdx 0, got %d", store.SelectedIdx)
	}
}

func TestStateStore_GetBlock(t *testing.T) {
	store := NewStateStore()

	store.AddBlock(Block{ID: "block-1", Command: "cmd1"})
	store.AddBlock(Block{ID: "block-2", Command: "cmd2"})

	block := store.GetBlock("block-1")
	if block == nil {
		t.Fatal("expected to find block-1")
	}
	if block.Command != "cmd1" {
		t.Errorf("expected cmd1, got %s", block.Command)
	}

	notFound := store.GetBlock("nonexistent")
	if notFound != nil {
		t.Error("should return nil for nonexistent block")
	}
}

func TestStateStore_GetRecentBlocks(t *testing.T) {
	store := NewStateStore()

	for i := 0; i < 5; i++ {
		store.AddBlock(Block{ID: string(rune('a' + i))})
	}

	recent := store.GetRecentBlocks(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(recent))
	}
	if recent[0].ID != "c" || recent[2].ID != "e" {
		t.Error("should return most recent blocks")
	}
}

func TestStateStore_GetBlocks(t *testing.T) {
	store := NewStateStore()

	store.AddBlock(Block{ID: "1"})
	store.AddBlock(Block{ID: "2"})

	blocks := store.GetBlocks()
	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}

	blocks[0].ID = "modified"
	if store.Blocks[0].ID == "modified" {
		t.Error("GetBlocks should return a copy")
	}
}

func TestStateStore_UpdateBlock(t *testing.T) {
	store := NewStateStore()

	store.AddBlock(Block{ID: "block-1", Output: ""})

	store.UpdateBlock("block-1", func(b *Block) {
		b.Output = "new output"
		b.ExitCode = 1
	})

	block := store.GetBlock("block-1")
	if block.Output != "new output" {
		t.Errorf("expected 'new output', got '%s'", block.Output)
	}
	if block.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", block.ExitCode)
	}
}

func TestStateStore_MaxBlocks(t *testing.T) {
	store := NewStateStore()
	store.MaxBlocks = 5

	for i := 0; i < 10; i++ {
		store.AddBlock(Block{ID: string(rune('0' + i))})
	}

	if len(store.Blocks) != 5 {
		t.Errorf("expected 5 blocks (MaxBlocks), got %d", len(store.Blocks))
	}

	if store.GetBlock("0") != nil {
		t.Error("block '0' should have been evicted")
	}
	if store.GetBlock("9") == nil {
		t.Error("block '9' should still exist")
	}
}

func TestStateStore_AddSuggestion(t *testing.T) {
	store := NewStateStore()

	store.AddSuggestion(Suggestion{
		ForBlockID:  "block-1",
		Title:       "Try this",
		Command:     "npm install",
		Explanation: "Missing dependencies",
	})

	if len(store.Suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(store.Suggestions))
	}
}

func TestStateStore_GetSuggestionsForBlock(t *testing.T) {
	store := NewStateStore()

	store.AddSuggestion(Suggestion{ForBlockID: "block-1", Title: "Sug1"})
	store.AddSuggestion(Suggestion{ForBlockID: "block-2", Title: "Sug2"})
	store.AddSuggestion(Suggestion{ForBlockID: "block-1", Title: "Sug3"})

	sugs := store.GetSuggestionsForBlock("block-1")
	if len(sugs) != 2 {
		t.Errorf("expected 2 suggestions for block-1, got %d", len(sugs))
	}
}

func TestStateStore_SuggestionLimit(t *testing.T) {
	store := NewStateStore()

	for i := 0; i < 15; i++ {
		store.AddSuggestion(Suggestion{
			ForBlockID: string(rune('a' + i)),
			Title:      "Suggestion",
		})
	}

	if len(store.Suggestions) != 10 {
		t.Errorf("expected 10 suggestions (limit), got %d", len(store.Suggestions))
	}
}

func TestStateStore_ClearBlocks(t *testing.T) {
	store := NewStateStore()

	store.AddBlock(Block{ID: "1"})
	store.AddBlock(Block{ID: "2"})

	store.ClearBlocks()

	if len(store.Blocks) != 0 {
		t.Errorf("expected 0 blocks after clear, got %d", len(store.Blocks))
	}
	if store.SelectedIdx != -1 {
		t.Errorf("expected SelectedIdx -1 after clear, got %d", store.SelectedIdx)
	}
}

func TestStateStore_LastError(t *testing.T) {
	store := NewStateStore()

	store.AddBlock(Block{ID: "1", ExitCode: 0})
	if store.LastError != nil {
		t.Error("LastError should be nil for successful command")
	}

	store.AddBlock(Block{ID: "2", ExitCode: 1})
	if store.LastError == nil {
		t.Fatal("LastError should be set for failed command")
	}
	if store.LastError.ID != "2" {
		t.Error("LastError should point to the failed block")
	}
}

func TestStateStore_SetDockerHealth(t *testing.T) {
	store := NewStateStore()

	health := infra.DockerHealth{
		Available: true,
		Containers: []infra.ContainerInfo{
			{ID: "abc123", Name: "test", State: "running"},
		},
	}

	store.SetDockerHealth(health)

	if !store.DockerHealth.Available {
		t.Error("DockerHealth.Available should be true")
	}
	if len(store.DockerHealth.Containers) != 1 {
		t.Error("DockerHealth.Containers should have 1 container")
	}
}

func TestStateStore_SetGPUStats(t *testing.T) {
	store := NewStateStore()

	stats := infra.GPUStats{
		Available: true,
	}

	store.SetGPUStats(stats)

	if !store.GPUStats.Available {
		t.Error("GPUStats.Available should be true")
	}
}

func TestStateStore_SetStarshipLine(t *testing.T) {
	store := NewStateStore()

	store.SetStarshipLine(" on main [!?]")

	if store.StarshipLine != " on main [!?]" {
		t.Errorf("StarshipLine not set correctly: %s", store.StarshipLine)
	}
}

func TestStateStore_SetCwd(t *testing.T) {
	store := NewStateStore()

	store.SetCwd("/home/user/project")

	if store.Cwd != "/home/user/project" {
		t.Errorf("Cwd not set correctly: %s", store.Cwd)
	}
}

func TestStateStore_GetContext(t *testing.T) {
	store := NewStateStore()

	store.SetCwd("/test")
	store.AddBlock(Block{ID: "1"})
	store.AddBlock(Block{ID: "2"})
	store.SetDockerHealth(infra.DockerHealth{
		Containers: []infra.ContainerInfo{{ID: "c1"}},
	})

	ctx := store.GetContext()

	if ctx["cwd"] != "/test" {
		t.Error("context should include cwd")
	}
	if ctx["recent_commands"] != 2 {
		t.Error("context should include recent_commands count")
	}
	if ctx["container_count"] != 1 {
		t.Error("context should include container_count")
	}
}

func TestStateStore_ConcurrentAccess(t *testing.T) {
	store := NewStateStore()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.AddBlock(Block{ID: string(rune('a' + (id % 26)))})
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.GetBlocks()
			_ = store.GetRecentBlocks(5)
		}()
	}

	wg.Wait()

	if len(store.Blocks) == 0 {
		t.Error("expected some blocks after concurrent access")
	}
}

func TestRebuildIndex(t *testing.T) {
	store := NewStateStore()
	store.MaxBlocks = 3

	store.AddBlock(Block{ID: "a"})
	store.AddBlock(Block{ID: "b"})
	store.AddBlock(Block{ID: "c"})
	store.AddBlock(Block{ID: "d"})

	if store.GetBlock("a") != nil {
		t.Error("block 'a' should have been evicted")
	}
	if store.GetBlock("d") == nil {
		t.Error("block 'd' should exist")
	}

	for id, idx := range store.blockIndex {
		if store.Blocks[idx].ID != id {
			t.Errorf("index mismatch: index[%s]=%d but Blocks[%d].ID=%s",
				id, idx, idx, store.Blocks[idx].ID)
		}
	}
}

package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dev-cli/internal/config"
	"dev-cli/internal/memory"
	"dev-cli/internal/storage"
)

// init wires the memory write-back callback. The storage package owns the
// callback variable; cmd/ owns the policy ("when MemPalace is enabled and
// write-back has not been opted out, store under dev-cli's wing as advice").
// This keeps storage import-free of memory.
func init() {
	storage.OnRunbookRecorded = func(cwd, issue, runbookID string, steps []storage.RecordedRunbookStep) {
		cfg := config.Load()
		if !cfg.MemPalaceEnabled || !cfg.MemPalaceWriteback {
			return
		}
		text := formatRunbookMemory(issue, steps)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = memory.Store(ctx, cfg, memory.StoreReq{
				Hall:   "advice",
				Cwd:    cwd,
				Text:   text,
				Source: "dev-cli/runbook:" + runbookID,
			})
		}()
	}
}

func formatRunbookMemory(issue string, steps []storage.RecordedRunbookStep) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Issue: %s\n\nResolution steps (%d):\n", strings.TrimSpace(issue), len(steps))
	for i, s := range steps {
		params := s.Parameters
		if len(params) > 240 {
			params = params[:240] + "…"
		}
		fmt.Fprintf(&b, "  %d. %s %s\n", i+1, s.ToolName, params)
	}
	return strings.TrimRight(b.String(), "\n")
}

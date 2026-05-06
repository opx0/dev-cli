package memory

import (
	"context"
	"strings"

	"dev-cli/internal/config"
)

// StoreReq is a write-back request for a single memory. Hall picks the slot
// in MemPalace's taxonomy: advice, problem, decision, snippet, fact, event,
// preference, milestone. Source is a free-form tag dev-cli uses to identify
// what produced this memory ("dev-cli/runbook", "dev-cli/explain", etc.) so
// users can filter or purge dev-cli-generated content without touching their
// curated entries.
type StoreReq struct {
	Hall   string
	Wing   string // "" => auto-derive from cwd
	Text   string
	Source string
	Cwd    string
}

// Store writes a memory back to MemPalace. Best-effort: returns nil silently
// when MemPalace is disabled, write-back is opt-out, the binary is missing,
// or the text is empty. Errors from the underlying CLI bubble up so callers
// can log them in verbose mode but should not surface them to the user.
//
// Recommended usage from a command's success path:
//
//	go func() {
//	    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	    defer cancel()
//	    _ = memory.Store(ctx, cfg, memory.StoreReq{...})
//	}()
func Store(ctx context.Context, cfg *config.Config, req StoreReq) error {
	if cfg == nil || !cfg.MemPalaceEnabled {
		return nil
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil
	}
	if !findBinaryAvailable() {
		return nil
	}

	wing := req.Wing
	if wing == "" {
		wing = cfg.MemPalaceWing
	}
	if wing == "" {
		wing = deriveWing(req.Cwd)
	}

	args := []string{"store", text}
	if wing != "" {
		args = append(args, "--wing", wing)
	}
	if req.Hall != "" {
		args = append(args, "--hall", req.Hall)
	}
	if req.Source != "" {
		args = append(args, "--source", req.Source)
	}

	ctx, cancel := withDefaultTimeout(ctx, storeTimeout)
	defer cancel()

	_, err := run(ctx, args)
	return err
}

// findBinaryAvailable is a small wrapper so Store can short-circuit without
// allocating an exec.Cmd just to discover the binary is absent.
func findBinaryAvailable() bool { return findBinary() != "" }

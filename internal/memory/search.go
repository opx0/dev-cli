package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dev-cli/internal/config"
)

// Result is one MemPalace hit decoded from `mempalace search --json` output.
// Field tags match the upstream `Hit.as_dict()` shape (db.py); unknown fields
// are tolerated so a future schema bump cannot break parsing.
type Result struct {
	ID     string  `json:"id"`
	Score  float64 `json:"score"`
	Wing   string  `json:"wing"`
	Room   string  `json:"room"`
	Hall   string  `json:"hall"`
	Text   string  `json:"text"`
	Source string  `json:"source"`
}

// SearchOpts controls a hybrid recall query. Empty fields fall back to config.
type SearchOpts struct {
	Limit int
	Wing  string // "" => use cfg.MemPalaceWing, then auto-derive from cwd
	Hall  string
	Cwd   string // for auto-wing derivation; "" => os.Getwd()
}

// Search runs a typed hybrid search. Returns an empty slice (not nil) when
// MemPalace is disabled, the binary is missing, or the call fails. Callers
// can therefore range over the result without nil-checks.
func Search(ctx context.Context, cfg *config.Config, query string, opts SearchOpts) ([]Result, error) {
	if cfg == nil || !cfg.MemPalaceEnabled || strings.TrimSpace(query) == "" {
		return []Result{}, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = cfg.MemPalaceLimit
	}
	if limit <= 0 {
		limit = defaultSearchSize
	}

	wing := opts.Wing
	if wing == "" {
		wing = cfg.MemPalaceWing
	}
	if wing == "" {
		wing = deriveWing(opts.Cwd)
	}

	hall := opts.Hall
	if hall == "" {
		hall = cfg.MemPalaceHall
	}

	args := []string{"search", query, "--limit", fmt.Sprintf("%d", limit), "--json"}
	if wing != "" {
		args = append(args, "--wing", wing)
	}
	if hall != "" {
		args = append(args, "--hall", hall)
	}

	ctx, cancel := withDefaultTimeout(ctx, defaultTimeout)
	defer cancel()

	out, err := run(ctx, args)
	if err != nil {
		return []Result{}, err
	}

	// Upstream sometimes prints rich-text headers before the JSON array when
	// stdout is a TTY; trim to the first '[' to be safe.
	trimmed := bytes_trimToJSON(out)
	if len(trimmed) == 0 {
		return []Result{}, nil
	}

	var hits []Result
	if err := json.Unmarshal(trimmed, &hits); err != nil {
		return []Result{}, fmt.Errorf("parse mempalace JSON: %w", err)
	}
	return hits, nil
}

// bytes_trimToJSON returns the substring starting at the first '[' (JSON array
// marker). Returns nil if no array is present. Implemented locally to avoid a
// `bytes` import name collision with the var-style declaration below.
func bytes_trimToJSON(b []byte) []byte {
	for i, c := range b {
		if c == '[' {
			return b[i:]
		}
	}
	return nil
}

package memory

import (
	"context"
	"fmt"
	"strings"

	"dev-cli/internal/config"
)

// BuildPromptContext returns a short memory block ready to prepend to an LLM
// prompt. It is read-only, best-effort, and safely returns ("", false) when
// MemPalace is disabled, the binary is missing, or no relevant hits exist.
//
// Kept as a thin shim over Search so existing call sites in cmd/ask.go and
// cmd/explain.go don't need to change. New call sites in fix/pr/review/gen
// also use this for prompt-injection convenience; for typed access the
// caller should use Search directly.
func BuildPromptContext(cfg *config.Config, query string) (string, bool) {
	if cfg == nil || !cfg.MemPalaceEnabled {
		return "", false
	}

	hits, err := Search(context.Background(), cfg, query, SearchOpts{})
	if err != nil || len(hits) == 0 {
		return "", false
	}

	var b strings.Builder
	b.WriteString("Relevant prior memory (use only if applicable):\n")
	for i, h := range hits {
		text := strings.TrimSpace(h.Text)
		if text == "" {
			continue
		}
		// Cap each hit to keep prompts compact; the full content is in the
		// index if the LLM ever needs more.
		if len(text) > 600 {
			text = text[:600] + "…"
		}
		fmt.Fprintf(&b, "  [%d] (%s/%s) %s\n", i+1, h.Wing, h.Hall, text)
	}
	return strings.TrimRight(b.String(), "\n"), true
}

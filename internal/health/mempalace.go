package health

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/config"
)

// CheckMemPalace verifies the MemPalace CLI when the user has opted in.
// When MemPalace is disabled, returns "ok" with a "disabled (opt-in)" note so
// the doctor output makes the feature discoverable without nagging users who
// have not enabled it.
func CheckMemPalace() CheckResult {
	cfg := config.Load()
	if !cfg.MemPalaceEnabled {
		return CheckResult{
			Name:    "MemPalace",
			Status:  "ok",
			Message: "disabled (set DEV_CLI_MEMPALACE_ENABLED=1 to enable hybrid recall)",
		}
	}

	bin := ""
	for _, name := range []string{"mempalace", "mp"} {
		if path, err := exec.LookPath(name); err == nil {
			bin = path
			break
		}
	}
	if bin == "" {
		return CheckResult{
			Name:    "MemPalace",
			Status:  "warn",
			Message: "enabled but binary not on PATH",
			FixCmd:  "pipx install mempalace",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--help").CombinedOutput()
	if err != nil {
		snippet := strings.TrimSpace(string(out))
		if len(snippet) > 80 {
			snippet = snippet[:80] + "…"
		}
		return CheckResult{
			Name:    "MemPalace",
			Status:  "warn",
			Message: "binary on PATH but help probe failed: " + snippet,
		}
	}

	return CheckResult{
		Name:    "MemPalace",
		Status:  "ok",
		Message: "available at " + bin,
	}
}

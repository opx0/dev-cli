package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dev-cli/internal/config"
)

// SaveErrorContext generates a markdown error report and writes it to
// the devlogs directory (~/.devlogs/last-error.md by default).
// Used by watch --opencode and export --save.
func SaveErrorContext(source, logs string) (string, error) {
	dir := config.Current.LogDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("# Error Context from dev-cli\n\n")
	sb.WriteString(fmt.Sprintf("**Source:** %s\n", source))
	sb.WriteString(fmt.Sprintf("**Detected:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("## Logs\n\n")
	sb.WriteString("```\n")
	sb.WriteString(logs)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Analyze these logs, identify the root cause of the error, and suggest fixes.\n")

	savePath := filepath.Join(dir, "last-error.md")
	if err := os.WriteFile(savePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	return savePath, nil
}

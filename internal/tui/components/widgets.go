package components

import (
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

type LogLine struct {
	Content string
	Level   string
}

func NewLogLine(content string) LogLine {
	line := LogLine{Content: content}
	upper := strings.ToUpper(content)
	switch {
	case strings.Contains(upper, "ERROR"), strings.Contains(upper, "ERR"):
		line.Level = "ERROR"
	case strings.Contains(upper, "WARN"):
		line.Level = "WARN"
	case strings.Contains(upper, "INFO"):
		line.Level = "INFO"
	case strings.Contains(upper, "DEBUG"), strings.Contains(upper, "TRACE"):
		line.Level = "DEBUG"
	}
	return line
}

func (l LogLine) Render() string {
	color := theme.Text
	switch l.Level {
	case "ERROR":
		color = theme.LogError
	case "WARN":
		color = theme.LogWarn
	case "INFO":
		color = theme.LogInfo
	case "DEBUG":
		color = theme.LogDebug
	}
	return lipgloss.NewStyle().Foreground(color).Render(l.Content)
}

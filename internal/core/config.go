package core

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/wordwrap"
)

type Config struct {
	OllamaURL       string
	OllamaModel     string
	PerplexityKey   string
	PerplexityModel string
	ForceLocalLLM   bool
	LogDir          string
}

func LoadConfig() *Config {
	cfg := &Config{
		OllamaURL:       "http://localhost:11434",
		OllamaModel:     "qwen2.5-coder:3b-instruct",
		PerplexityModel: "sonar-pro",
		ForceLocalLLM:   false,
	}

	if val := os.Getenv("DEV_CLI_OLLAMA_URL"); val != "" {
		cfg.OllamaURL = val
	}
	if val := os.Getenv("DEV_CLI_OLLAMA_MODEL"); val != "" {
		cfg.OllamaModel = val
	}
	if val := os.Getenv("DEV_CLI_PERPLEXITY_KEY"); val != "" {
		cfg.PerplexityKey = val
	} else if val := os.Getenv("PERPLEXITY_API_KEY"); val != "" {
		cfg.PerplexityKey = val
	}
	if val := os.Getenv("DEV_CLI_PERPLEXITY_MODEL"); val != "" {
		cfg.PerplexityModel = val
	}
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" {
		cfg.ForceLocalLLM = true
	}
	if val := os.Getenv("DEV_CLI_LOG_DIR"); val != "" {
		cfg.LogDir = val
	} else {
		home, _ := os.UserHomeDir()
		cfg.LogDir = filepath.Join(home, ".devlogs")
	}

	return cfg
}

func (c *Config) IsWebSearchEnabled() bool {
	return !c.ForceLocalLLM && c.PerplexityKey != ""
}

var CurrentConfig = LoadConfig()

func CutLine(line string, start, end int) string {
	if start >= end {
		return ""
	}
	return ansi.Cut(line, start, end)
}

func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return wordwrap.String(text, width)
}

func TruncateWithEllipsis(line string, width int) string {
	lineWidth := ansi.StringWidth(line)
	if lineWidth <= width {
		return line
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return ansi.Cut(line, 0, width-3) + "..."
}

func StringWidth(s string) int {
	return ansi.StringWidth(s)
}

func ProcessLinesForViewport(lines []string, width, xOffset int, wrapMode bool) []string {
	if len(lines) == 0 {
		return lines
	}

	if wrapMode {
		var result []string
		for _, line := range lines {
			if ansi.StringWidth(line) <= width {
				result = append(result, line)
			} else {
				wrapped := wordwrap.String(line, width)
				result = append(result, strings.Split(wrapped, "\n")...)
			}
		}
		return result
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		lineWidth := ansi.StringWidth(line)
		if lineWidth <= width || xOffset == 0 && lineWidth <= width {
			result[i] = line
		} else {
			result[i] = ansi.Cut(line, xOffset, xOffset+width)
		}
	}
	return result
}

func MaxLineWidth(lines []string) int {
	maxWidth := 0
	for _, line := range lines {
		w := ansi.StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func TruncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen-20] + "\n...[truncated]..."
}

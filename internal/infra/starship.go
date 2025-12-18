package infra

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// StarshipPrompt holds the rendered starship prompt
type StarshipPrompt struct {
	Available bool
	Raw       string   // Raw prompt output
	Clean     string   // ANSI-stripped version
	Segments  []string // Parsed segments
}

// GetStarshipPrompt runs starship and returns the prompt
func GetStarshipPrompt() StarshipPrompt {
	prompt := StarshipPrompt{}

	// Check if starship is available
	path, err := exec.LookPath("starship")
	if err != nil || path == "" {
		return prompt
	}
	prompt.Available = true

	// Get the prompt
	cmd := exec.Command("starship", "prompt")
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return prompt
	}

	prompt.Raw = string(out)
	prompt.Clean = stripANSI(prompt.Raw)

	// Parse segments (split by common separators)
	prompt.Segments = parseStarshipSegments(prompt.Clean)

	return prompt
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences including %{...%} wrappers
	re := regexp.MustCompile(`\x1b\[[0-9;]*m|%\{[^}]*\}`)
	clean := re.ReplaceAllString(s, "")
	// Remove double spaces and trim
	clean = strings.Join(strings.Fields(clean), " ")
	return strings.TrimSpace(clean)
}

// parseStarshipSegments breaks the prompt into meaningful parts
func parseStarshipSegments(clean string) []string {
	// Split by common starship separators
	separators := []string{" on ", " in ", " via ", " is ", " took "}

	parts := []string{clean}
	for _, sep := range separators {
		var newParts []string
		for _, part := range parts {
			split := strings.Split(part, sep)
			for i, s := range split {
				s = strings.TrimSpace(s)
				if s != "" {
					if i > 0 {
						// Preserve separator info
						newParts = append(newParts, sep[1:len(sep)-1]+" "+s)
					} else {
						newParts = append(newParts, s)
					}
				}
			}
		}
		parts = newParts
	}

	// Remove prompt character
	var filtered []string
	for _, p := range parts {
		p = strings.TrimLeft(p, "❯> ")
		p = strings.TrimSpace(p)
		if p != "" && p != "❯" && p != ">" {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// GetStarshipStatusLine returns a compact status line for TUI
func GetStarshipStatusLine() string {
	prompt := GetStarshipPrompt()
	if !prompt.Available {
		return ""
	}

	// Clean and format for TUI status bar
	// Remove the final prompt character (❯ or >) and extra newlines
	line := prompt.Clean
	line = strings.TrimRight(line, "❯> \n\r")
	line = strings.TrimSpace(line)

	return line
}

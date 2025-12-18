package infra

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type StarshipPrompt struct {
	Available bool
	Raw       string   // Raw prompt output
	Clean     string   // ANSI-stripped version
	Segments  []string // Parsed segments
}

func GetStarshipPrompt() StarshipPrompt {
	prompt := StarshipPrompt{}

	path, err := exec.LookPath("starship")
	if err != nil || path == "" {
		return prompt
	}
	prompt.Available = true

	cmd := exec.Command("starship", "prompt")
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return prompt
	}

	prompt.Raw = string(out)
	prompt.Clean = stripANSI(prompt.Raw)

	prompt.Segments = parseStarshipSegments(prompt.Clean)

	return prompt
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m|%\{[^}]*\}`)
	clean := re.ReplaceAllString(s, "")
	clean = strings.Join(strings.Fields(clean), " ")
	return strings.TrimSpace(clean)
}

func parseStarshipSegments(clean string) []string {
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
						newParts = append(newParts, sep[1:len(sep)-1]+" "+s)
					} else {
						newParts = append(newParts, s)
					}
				}
			}
		}
		parts = newParts
	}

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

func GetStarshipStatusLine() string {
	prompt := GetStarshipPrompt()
	if !prompt.Available {
		return ""
	}

	line := prompt.Clean
	line = strings.TrimRight(line, "❯> \n\r")
	line = strings.TrimSpace(line)

	return line
}

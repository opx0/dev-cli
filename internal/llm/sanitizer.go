package llm

import (
	"regexp"
	"strings"
)

type Sanitizer struct {
	patterns []*secretPattern
}

type secretPattern struct {
	regex       *regexp.Regexp
	replacement string
	name        string
}

func DefaultSanitizer() *Sanitizer {
	return &Sanitizer{
		patterns: []*secretPattern{
			{
				regex:       regexp.MustCompile(`(?i)(api[_-]?key|apikey)[=:]["']?([a-zA-Z0-9_\-]{20,})["']?`),
				replacement: `$1=[REDACTED_API_KEY]`,
				name:        "API Key",
			},
			{
				regex:       regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9_\-\.]+)`),
				replacement: `Bearer [REDACTED_TOKEN]`,
				name:        "Bearer Token",
			},
			{
				regex:       regexp.MustCompile(`(?i)(AKIA|ABIA|ACCA|ASIA)[A-Z0-9]{16}`),
				replacement: `[REDACTED_AWS_KEY]`,
				name:        "AWS Access Key",
			},
			{
				regex:       regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?access[_-]?key)[=:]["']?([a-zA-Z0-9/+=]{40})["']?`),
				replacement: `$1=[REDACTED_AWS_SECRET]`,
				name:        "AWS Secret Key",
			},
			{
				regex:       regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----[\s\S]*?-----END\s+(RSA\s+)?PRIVATE KEY-----`),
				replacement: `[REDACTED_PRIVATE_KEY]`,
				name:        "Private Key",
			},
			{
				regex:       regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
				replacement: `[REDACTED_GITHUB_TOKEN]`,
				name:        "GitHub PAT",
			},
			{
				regex:       regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token)[=:]["']?([^\s"']{8,})["']?`),
				replacement: `$1=[REDACTED]`,
				name:        "Password/Secret",
			},
			{
				regex:       regexp.MustCompile(`(?i)(mongodb|postgresql|mysql|redis)://[^:]+:([^@]+)@`),
				replacement: `$1://[user]:[REDACTED]@`,
				name:        "Database Password",
			},
			{
				regex:       regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),
				replacement: `[REDACTED_JWT]`,
				name:        "JWT Token",
			},
			{
				regex:       regexp.MustCompile(`xox[baprs]-[a-zA-Z0-9-]+`),
				replacement: `[REDACTED_SLACK_TOKEN]`,
				name:        "Slack Token",
			},
		},
	}
}

func (s *Sanitizer) Sanitize(input string) string {
	result := input
	for _, pattern := range s.patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}
	return result
}

func (s *Sanitizer) SanitizeWithReport(input string) (sanitized string, found []string) {
	result := input
	for _, pattern := range s.patterns {
		if pattern.regex.MatchString(result) {
			found = append(found, pattern.name)
			result = pattern.regex.ReplaceAllString(result, pattern.replacement)
		}
	}
	return result, found
}

func (s *Sanitizer) ContainsSecrets(input string) bool {
	for _, pattern := range s.patterns {
		if pattern.regex.MatchString(input) {
			return true
		}
	}
	return false
}

var globalSanitizer = DefaultSanitizer()

func SanitizeForLLM(input string) string {
	return globalSanitizer.Sanitize(input)
}

func MaskEnvVars(input string) string {
	sensitiveVars := regexp.MustCompile(`(?i)(export\s+)?(API_KEY|SECRET|PASSWORD|TOKEN|PRIVATE_KEY|AWS_SECRET)[=]["']?([^\s"'\n]+)["']?`)
	return sensitiveVars.ReplaceAllString(input, `$1$2=[REDACTED]`)
}

func SanitizeOutput(output string) string {
	result := globalSanitizer.Sanitize(output)
	result = MaskEnvVars(result)
	return result
}

func (s *Sanitizer) AddPattern(name, pattern, replacement string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	s.patterns = append(s.patterns, &secretPattern{
		regex:       re,
		replacement: replacement,
		name:        name,
	})
	return nil
}

func (s *Sanitizer) PatternCount() int {
	return len(s.patterns)
}

func TruncateForLLM(input string, maxLen int) string {
	if len(input) <= maxLen {
		return input
	}
	half := (maxLen - 20) / 2
	return input[:half] + "\n...[truncated]...\n" + input[len(input)-half:]
}

func PrepareForLLM(input string, maxLen int) string {
	sanitized := SanitizeForLLM(input)
	if maxLen > 0 {
		sanitized = TruncateForLLM(sanitized, maxLen)
	}
	return strings.TrimSpace(sanitized)
}

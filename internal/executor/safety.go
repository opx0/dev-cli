package executor

import "strings"

// dangerousPatterns is the list of shell patterns that could cause data loss.
var dangerousPatterns = []string{
	"rm -rf",
	"rm -r /",
	"dd if=",
	"mkfs",
	"> /dev/",
	"chmod 777",
	":(){ :|:& };:",
}

// IsDangerousCommand checks whether cmd contains any dangerous pattern.
// Returns the matched pattern or empty string.
func IsDangerousCommand(cmd string) string {
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return pattern
		}
	}
	return ""
}

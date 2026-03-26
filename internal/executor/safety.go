package executor

import "strings"

// DangerousPatterns is the canonical list of shell patterns that could cause data loss.
// This is the shared source of truth used by both the executor and workflow packages.
var DangerousPatterns = []string{
	// File system destructive operations
	"rm -rf",
	"rm -r /",
	"dd if=",
	"mkfs",
	"> /dev/",
	"chmod 777",

	// Fork bomb
	":(){ :|:& };:",

	// Database destructive operations
	"drop database",
	"drop table",
	"truncate table",
	"delete from",

	// Git destructive operations
	"git reset --hard",
	"git clean -fdx",

	// Docker destructive operations
	"docker system prune",
}

// IsDangerousCommand checks whether cmd contains any dangerous pattern.
// Returns the matched pattern or empty string.
func IsDangerousCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	for _, pattern := range DangerousPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return pattern
		}
	}
	return ""
}

// IsDangerous is an alias for IsDangerousCommand that returns a bool.
func IsDangerous(cmd string) bool {
	return IsDangerousCommand(cmd) != ""
}

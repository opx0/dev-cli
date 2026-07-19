package executor

import (
	"os"
	"path/filepath"
	"strings"
)

// DangerousPatterns is the canonical list of shell patterns that could cause data loss.
// This is the shared source of truth for command and file safety checks.
var DangerousPatterns = []string{
	// File system destructive operations
	"rm -rf",
	"rm -r /",
	"rm -fr",
	"dd if=",
	"mkfs",
	"> /dev/",
	"chmod 777",
	"chmod -R 777",
	"chown -R",
	"shred",

	// Fork bomb and system crashes
	":(){ :|:& };:",
	"shutdown",
	"reboot",
	"init 0",
	"init 6",
	"halt",
	"poweroff",

	// Database destructive operations
	"drop database",
	"drop table",
	"truncate table",
	"delete from",
	"drop schema",

	// Git destructive operations
	"git reset --hard",
	"git clean -fdx",
	"git clean -fd",
	"git push --force",
	"git push -f",
	"git push origin --force",
	"git push origin -f",
	"git checkout -- .",
	"git stash drop",
	"git stash clear",
	"git branch -D",
	"git reflog expire",
	"git gc --prune",

	// Docker destructive operations
	"docker system prune",
	"docker volume prune",
	"docker container prune",
	"docker image prune -a",
	"docker rm -f",
	"docker rmi -f",
	"docker stop $(docker ps",
	"docker kill $(docker ps",

	// Kubernetes destructive operations
	"kubectl delete namespace",
	"kubectl delete ns",
	"kubectl delete --all",
	"kubectl delete pods --all",
	"kubectl delete deployment --all",
	"helm uninstall",
	"helm delete",

	// Package manager destructive
	"pip uninstall -y",
	"npm uninstall -g",
	"apt-get remove --purge",
	"apt-get autoremove",

	// Network destructive
	"iptables -F",
	"iptables --flush",
	"ufw disable",

	// Environment/config destructive
	"unset PATH",
	"export PATH=",
}

// SensitiveFilePatterns are file patterns that should never be read or written by the agent.
// These typically contain secrets, credentials, or sensitive configuration.
var SensitiveFilePatterns = []string{
	// Environment and secrets
	".env",
	".env.local",
	".env.production",
	".env.development",
	"*.env",

	// Private keys and certificates
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",
	"*.jks",
	"id_rsa",
	"id_ed25519",
	"id_ecdsa",
	"id_dsa",

	// Credentials and tokens
	"credentials.json",
	"credentials.yaml",
	"credentials.yml",
	"secrets.json",
	"secrets.yaml",
	"secrets.yml",
	"*_credentials*",
	"*_secrets*",
	".netrc",
	".npmrc",
	".pypirc",

	// Cloud provider configs
	"*.tfvars",
	"terraform.tfstate",
	"terraform.tfstate.backup",
	".aws/credentials",
	".azure/credentials",
	"gcloud/credentials",
	"service-account*.json",
	"serviceaccount*.json",

	// Database configs
	"database.yml",
	"database.yaml",
	"db.json",
	"*.sqlite",
	"*.db",

	// Auth configs
	".htpasswd",
	"passwd",
	"shadow",
	"authorized_keys",
	"known_hosts",

	// Application secrets
	"master.key",
	"secret_key_base",
	"jwt_secret",
	"api_key*",
	"apikey*",

	// History files (may contain secrets)
	".bash_history",
	".zsh_history",
	".python_history",
	".mysql_history",
	".psql_history",
}

// SensitiveDirectories are directories that should be treated with extra caution.
var SensitiveDirectories = []string{
	".git",
	".ssh",
	".gnupg",
	".aws",
	".azure",
	".config/gcloud",
	".kube",
	"node_modules",
	"venv",
	".venv",
	"__pycache__",
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

// IsSensitiveFile checks if a file path matches any sensitive file pattern.
// Returns the matched pattern or empty string.
func IsSensitiveFile(path string) string {
	// Normalize path
	path = filepath.Clean(path)
	base := filepath.Base(path)
	baseLower := strings.ToLower(base)
	pathLower := strings.ToLower(path)

	for _, pattern := range SensitiveFilePatterns {
		patternLower := strings.ToLower(pattern)

		// Check exact match on basename
		if baseLower == patternLower {
			return pattern
		}

		// Check glob pattern match
		if strings.Contains(pattern, "*") {
			// Simple glob matching
			if matchGlob(patternLower, baseLower) {
				return pattern
			}
			// Also check against full path for patterns like ".aws/credentials"
			if matchGlob(patternLower, pathLower) {
				return pattern
			}
		}

		// Check if pattern is in path (for paths like ".aws/credentials")
		if strings.Contains(pathLower, patternLower) {
			return pattern
		}
	}
	return ""
}

// IsSensitiveDirectory checks if a path is within a sensitive directory.
// Returns the matched directory or empty string.
func IsSensitiveDirectory(path string) string {
	path = filepath.Clean(path)
	pathLower := strings.ToLower(path)

	for _, dir := range SensitiveDirectories {
		dirLower := strings.ToLower(dir)
		// Check if path contains the sensitive directory
		if strings.Contains(pathLower, string(filepath.Separator)+dirLower+string(filepath.Separator)) ||
			strings.HasPrefix(pathLower, dirLower+string(filepath.Separator)) ||
			strings.HasSuffix(pathLower, string(filepath.Separator)+dirLower) ||
			pathLower == dirLower {
			return dir
		}
	}
	return ""
}

// matchGlob provides simple glob matching for patterns with * wildcards.
func matchGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// ResolvePath returns an absolute path with existing symlinks resolved. For a
// path that does not exist yet, it resolves the nearest existing parent.
func ResolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	parent := filepath.Dir(abs)
	for {
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			rel, err := filepath.Rel(parent, abs)
			if err != nil {
				return "", err
			}
			return filepath.Join(resolved, rel), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return abs, nil
		}
		parent = next
	}
}

// IsPathWithin reports whether path is scope itself or one of its descendants.
func IsPathWithin(scope, path string) bool {
	rel, err := filepath.Rel(scope, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// SafetyCheck represents the result of a safety check.
type SafetyCheck struct {
	IsSafe      bool
	Reason      string
	MatchedRule string
	Severity    string // "warning", "danger", "critical"
}

// CheckCommandSafety performs a comprehensive safety check on a command.
func CheckCommandSafety(cmd string) SafetyCheck {
	if pattern := IsDangerousCommand(cmd); pattern != "" {
		severity := "danger"
		// Some patterns are critical (system-level)
		critical := []string{"rm -rf /", "dd if=", "mkfs", "shutdown", "reboot", ":(){ :|:& };:"}
		for _, c := range critical {
			if strings.Contains(strings.ToLower(pattern), c) {
				severity = "critical"
				break
			}
		}
		return SafetyCheck{
			IsSafe:      false,
			Reason:      "Command matches dangerous pattern",
			MatchedRule: pattern,
			Severity:    severity,
		}
	}
	return SafetyCheck{IsSafe: true}
}

// CheckFileSafety performs a comprehensive safety check on a file path.
func CheckFileSafety(path string) SafetyCheck {
	if pattern := IsSensitiveFile(path); pattern != "" {
		return SafetyCheck{
			IsSafe:      false,
			Reason:      "File matches sensitive pattern",
			MatchedRule: pattern,
			Severity:    "danger",
		}
	}
	if dir := IsSensitiveDirectory(path); dir != "" {
		return SafetyCheck{
			IsSafe:      false,
			Reason:      "Path is within sensitive directory",
			MatchedRule: dir,
			Severity:    "warning",
		}
	}
	return SafetyCheck{IsSafe: true}
}

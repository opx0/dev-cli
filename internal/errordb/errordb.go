// Package errordb provides a built-in database of common error patterns and
// their known fixes. This is consulted before calling the LLM so that trivial,
// well-known errors get instant answers without network/GPU latency.
package errordb

import "strings"

// Pattern describes a single recognizable error and its known fix.
type Pattern struct {
	Category    string // e.g. "npm", "docker", "git", "k8s", "go", "python"
	Match       string // Substring to look for in command+output (case-insensitive)
	Explanation string // Human-readable one-liner
	Fix         string // Exact shell command to run (empty = no quick fix)
}

// knownPatterns is the built-in pattern set. Ordered by specificity (most
// specific first within each category).
var knownPatterns = []Pattern{
	// ── npm / Node ───────────────────────────────────────────────────────
	{Category: "npm", Match: "enoent open 'package.json'", Explanation: "Missing package.json in current directory", Fix: "npm init -y"},
	{Category: "npm", Match: "enoent", Explanation: "File or directory not found by npm", Fix: ""},
	{Category: "npm", Match: "cannot find module", Explanation: "Node module not installed", Fix: "npm install"},
	{Category: "npm", Match: "eresolve unable to resolve dependency", Explanation: "Dependency version conflict", Fix: "npm install --legacy-peer-deps"},
	{Category: "npm", Match: "eacces permission denied", Explanation: "Permission denied writing to npm directories", Fix: "sudo chown -R $(whoami) ~/.npm"},
	{Category: "npm", Match: "err_socket_timeout", Explanation: "npm registry request timed out", Fix: "npm cache clean --force && npm install"},
	{Category: "npm", Match: "node-gyp", Explanation: "Native module build failed — missing build tools", Fix: "sudo pacman -S base-devel python"},
	{Category: "npm", Match: "engine \"node\"", Explanation: "Node.js version mismatch", Fix: "nvm use --lts"},

	// ── Docker ───────────────────────────────────────────────────────────
	{Category: "docker", Match: "permission denied while trying to connect to the docker daemon", Explanation: "User not in docker group", Fix: "sudo usermod -aG docker $USER && newgrp docker"},
	{Category: "docker", Match: "cannot connect to the docker daemon", Explanation: "Docker daemon is not running", Fix: "sudo systemctl start docker"},
	{Category: "docker", Match: "port is already allocated", Explanation: "Port already in use by another container or process", Fix: ""},
	{Category: "docker", Match: "no space left on device", Explanation: "Docker disk space exhausted", Fix: "docker system prune -f"},
	{Category: "docker", Match: "manifest unknown", Explanation: "Docker image tag not found in registry", Fix: ""},
	{Category: "docker", Match: "name is already in use by container", Explanation: "Container with that name already exists", Fix: ""},
	{Category: "docker", Match: "oci runtime create failed", Explanation: "Container runtime error (often missing entrypoint or bad image)", Fix: ""},
	{Category: "docker", Match: "network not found", Explanation: "Docker network does not exist", Fix: "docker network create"},

	// ── Git ──────────────────────────────────────────────────────────────
	{Category: "git", Match: "not a git repository", Explanation: "Not inside a git repository", Fix: "git init"},
	{Category: "git", Match: "failed to push some refs", Explanation: "Remote has commits you don't have locally", Fix: "git pull --rebase && git push"},
	{Category: "git", Match: "your branch is behind", Explanation: "Local branch is behind remote", Fix: "git pull"},
	{Category: "git", Match: "merge conflict", Explanation: "Git merge conflict — files need manual resolution", Fix: ""},
	{Category: "git", Match: "nothing to commit", Explanation: "No staged changes to commit", Fix: ""},
	{Category: "git", Match: "please commit your changes or stash them", Explanation: "Uncommitted changes blocking operation", Fix: "git stash"},
	{Category: "git", Match: "fatal: authentication failed", Explanation: "Git authentication failed", Fix: ""},
	{Category: "git", Match: "permission denied (publickey)", Explanation: "SSH key not authorized for this repo", Fix: "ssh-add ~/.ssh/id_ed25519"},

	// ── Kubernetes ───────────────────────────────────────────────────────
	{Category: "k8s", Match: "the connection to the server was refused", Explanation: "Cannot connect to Kubernetes cluster", Fix: "kubectl config current-context"},
	{Category: "k8s", Match: "error: the server doesn't have a resource type", Explanation: "Unknown Kubernetes resource type (CRD not installed?)", Fix: ""},
	{Category: "k8s", Match: "forbidden: user", Explanation: "RBAC permission denied", Fix: ""},
	{Category: "k8s", Match: "imagepullbackoff", Explanation: "Kubernetes cannot pull container image", Fix: ""},
	{Category: "k8s", Match: "crashloopbackoff", Explanation: "Pod keeps crashing and restarting", Fix: "kubectl logs <pod-name> --previous"},
	{Category: "k8s", Match: "evicted", Explanation: "Pod evicted due to resource limits (disk/memory)", Fix: ""},

	// ── Go ───────────────────────────────────────────────────────────────
	{Category: "go", Match: "cannot find package", Explanation: "Go package not found — may need go mod tidy", Fix: "go mod tidy"},
	{Category: "go", Match: "go.sum mismatch", Explanation: "Go checksum mismatch — module may have been tampered with", Fix: "go clean -modcache && go mod download"},
	{Category: "go", Match: "missing go.sum entry", Explanation: "Module missing from go.sum", Fix: "go mod tidy"},
	{Category: "go", Match: "cannot find main module", Explanation: "Not in a Go module directory", Fix: "go mod init"},
	{Category: "go", Match: "undefined:", Explanation: "Undefined symbol — likely missing import or typo", Fix: ""},

	// ── Python ───────────────────────────────────────────────────────────
	{Category: "python", Match: "modulenotfounderror", Explanation: "Python module not installed", Fix: "pip install"},
	{Category: "python", Match: "no module named", Explanation: "Python module not installed", Fix: "pip install"},
	{Category: "python", Match: "syntaxerror", Explanation: "Python syntax error", Fix: ""},
	{Category: "python", Match: "permissionerror: [errno 13]", Explanation: "Python permission denied — use venv or --user", Fix: "pip install --user"},

	// ── System / General ─────────────────────────────────────────────────
	{Category: "system", Match: "command not found", Explanation: "Command not installed", Fix: ""},
	{Category: "system", Match: "permission denied", Explanation: "Permission denied — may need sudo or different permissions", Fix: ""},
	{Category: "system", Match: "no such file or directory", Explanation: "File or directory does not exist", Fix: ""},
	{Category: "system", Match: "address already in use", Explanation: "Port already in use by another process", Fix: ""},
	{Category: "system", Match: "connection refused", Explanation: "Service is not running or not accepting connections", Fix: ""},
	{Category: "system", Match: "disk quota exceeded", Explanation: "Disk quota or space exhausted", Fix: "df -h"},
	{Category: "system", Match: "too many open files", Explanation: "File descriptor limit reached", Fix: "ulimit -n 65535"},
	{Category: "system", Match: "killed", Explanation: "Process killed — likely OOM (out of memory)", Fix: ""},
}

// Lookup searches the error database for a matching pattern against the combined
// command + output text. Returns (pattern, true) on match, (zero, false) otherwise.
func Lookup(command, output string) (Pattern, bool) {
	combined := strings.ToLower(command + " " + output)
	for _, p := range knownPatterns {
		if strings.Contains(combined, p.Match) {
			return p, true
		}
	}
	return Pattern{}, false
}

// LookupAll returns all matching patterns (there may be multiple).
func LookupAll(command, output string) []Pattern {
	combined := strings.ToLower(command + " " + output)
	var matches []Pattern
	for _, p := range knownPatterns {
		if strings.Contains(combined, p.Match) {
			matches = append(matches, p)
		}
	}
	return matches
}

// Categories returns the list of supported error categories.
func Categories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, p := range knownPatterns {
		if !seen[p.Category] {
			seen[p.Category] = true
			cats = append(cats, p.Category)
		}
	}
	return cats
}

// Count returns the total number of known patterns.
func Count() int {
	return len(knownPatterns)
}

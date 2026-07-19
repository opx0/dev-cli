package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCappedBuffer(t *testing.T) {
	buffer := newCappedBuffer()
	payload := make([]byte, maxCapturedOutput+10)
	written, err := buffer.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	if written != len(payload) {
		t.Fatalf("writer reported %d bytes, want %d", written, len(payload))
	}
	if !strings.HasSuffix(buffer.String(), "... output truncated") {
		t.Fatal("capped output did not report truncation")
	}
}

func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantDang bool
	}{
		// Dangerous commands
		{"rm -rf", "rm -rf /tmp/test", true},
		{"git force push", "git push --force origin main", true},
		{"git push -f", "git push -f origin main", true},
		{"git reset hard", "git reset --hard HEAD~1", true},
		{"drop database", "DROP DATABASE users;", true},
		{"docker prune", "docker system prune -a", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"kubectl delete all", "kubectl delete --all pods", true},

		// Safe commands
		{"ls", "ls -la", false},
		{"git status", "git status", false},
		{"git push normal", "git push origin main", false},
		{"docker ps", "docker ps -a", false},
		{"cat file", "cat /etc/hosts", false},
		{"go build", "go build ./...", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDangerous(tt.command)
			if got != tt.wantDang {
				t.Errorf("IsDangerous(%q) = %v, want %v", tt.command, got, tt.wantDang)
			}
		})
	}
}

func TestIsSensitiveFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantSen bool
	}{
		// Sensitive files
		{".env", "/app/.env", true},
		{".env.local", "/project/.env.local", true},
		{"id_rsa", "/home/user/.ssh/id_rsa", true},
		{"pem file", "/etc/ssl/private/server.pem", true},
		{"key file", "/etc/ssl/private/server.key", true},
		{"credentials.json", "/app/credentials.json", true},
		{"credentials wildcard", "/app/prod_credentials_backup", true},
		{"secrets wildcard", "/app/team_secrets_old", true},
		{"tfvars", "/terraform/prod.tfvars", true},
		{"aws credentials", "/home/user/.aws/credentials", true},
		{"master.key", "/app/config/master.key", true},

		// Non-sensitive files
		{"go file", "/app/main.go", false},
		{"readme", "/project/README.md", false},
		{"config", "/app/config.yaml", false},
		{"dockerfile", "/app/Dockerfile", false},
		{"makefile", "/app/Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSensitiveFile(tt.path)
			if (got != "") != tt.wantSen {
				t.Errorf("IsSensitiveFile(%q) = %q, wantSensitive = %v", tt.path, got, tt.wantSen)
			}
		})
	}
}

func TestResolvePathResolvesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "harmless.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolvePath(link)
	if err != nil {
		t.Fatal(err)
	}
	if CheckFileSafety(resolved).IsSafe {
		t.Fatal("symlink to sensitive file was considered safe")
	}
}

func TestIsPathWithin(t *testing.T) {
	if !IsPathWithin("/work/app", "/work/app/sub/file.go") {
		t.Fatal("child path should be inside scope")
	}
	if IsPathWithin("/work/app", "/work/application/file.go") {
		t.Fatal("sibling-prefix path escaped scope")
	}
}

func TestIsSensitiveDirectory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantSen bool
	}{
		// Sensitive directories
		{".git", "/project/.git/config", true},
		{".ssh", "/home/user/.ssh/config", true},
		{".aws", "/home/user/.aws/config", true},
		{".gnupg", "/home/user/.gnupg/pubring.gpg", true},

		// Non-sensitive directories
		{"src", "/project/src/main.go", false},
		{"app", "/home/user/app/config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSensitiveDirectory(tt.path)
			if (got != "") != tt.wantSen {
				t.Errorf("IsSensitiveDirectory(%q) = %q, wantSensitive = %v", tt.path, got, tt.wantSen)
			}
		})
	}
}

func TestCheckCommandSafety(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		wantSafe     bool
		wantSeverity string
	}{
		{"safe command", "git status", true, ""},
		{"danger rm -rf", "rm -rf /tmp", false, "danger"},
		{"critical fork bomb", ":(){ :|:& };:", false, "critical"},
		{"danger git force", "git push --force", false, "danger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := CheckCommandSafety(tt.command)
			if check.IsSafe != tt.wantSafe {
				t.Errorf("CheckCommandSafety(%q).IsSafe = %v, want %v", tt.command, check.IsSafe, tt.wantSafe)
			}
			if !tt.wantSafe && check.Severity != tt.wantSeverity {
				t.Errorf("CheckCommandSafety(%q).Severity = %q, want %q", tt.command, check.Severity, tt.wantSeverity)
			}
		})
	}
}

func TestCheckFileSafety(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantSafe bool
	}{
		{"safe go file", "/app/main.go", true},
		{"unsafe .env", "/app/.env", false},
		{"unsafe key", "/app/server.key", false},
		{"unsafe ssh dir", "/home/user/.ssh/id_rsa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := CheckFileSafety(tt.path)
			if check.IsSafe != tt.wantSafe {
				t.Errorf("CheckFileSafety(%q).IsSafe = %v, want %v", tt.path, check.IsSafe, tt.wantSafe)
			}
		})
	}
}

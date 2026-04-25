package errordb

import "testing"

func TestLookup_KnownPatterns(t *testing.T) {
	tests := []struct {
		command  string
		output   string
		wantHit  bool
		wantCat  string
	}{
		{"npm install", "ENOENT open 'package.json'", true, "npm"},
		{"docker ps", "Cannot connect to the Docker daemon", true, "docker"},
		{"git push", "failed to push some refs", true, "git"},
		{"kubectl get pods", "The connection to the server was refused", true, "k8s"},
		{"go build ./...", "cannot find package", true, "go"},
		{"python app.py", "ModuleNotFoundError: No module named 'flask'", true, "python"},
		{"ls /nonexistent", "No such file or directory", true, "system"},
		{"something random", "everything is fine", false, ""},
	}

	for _, tt := range tests {
		p, ok := Lookup(tt.command, tt.output)
		if ok != tt.wantHit {
			t.Errorf("Lookup(%q, %q): got hit=%v, want %v", tt.command, tt.output, ok, tt.wantHit)
		}
		if ok && p.Category != tt.wantCat {
			t.Errorf("Lookup(%q, %q): got category=%q, want %q", tt.command, tt.output, p.Category, tt.wantCat)
		}
	}
}

func TestLookupAll_MultipleMatches(t *testing.T) {
	// "permission denied" could match both docker and system patterns
	matches := LookupAll("docker run", "permission denied while trying to connect to the docker daemon")
	if len(matches) < 2 {
		t.Errorf("LookupAll: expected at least 2 matches, got %d", len(matches))
	}
}

func TestCount(t *testing.T) {
	if c := Count(); c < 40 {
		t.Errorf("Count() = %d, want at least 40 patterns", c)
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) < 6 {
		t.Errorf("Categories() = %d, want at least 6", len(cats))
	}
}

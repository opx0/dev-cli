package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExplain_CommandNotFound(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			return
		}

		// Verify request
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}

		if req.Stream != false {
			t.Error("expected stream to be false")
		}

		// Return mock response
		resp := generateResponse{
			Response: `{"explanation": "Command 'asdfnotfound' was not found in PATH", "fix": ""}`,
			Done:     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server
	client := &Client{
		baseURL:    server.URL,
		model:      "test-model",
		httpClient: http.DefaultClient,
	}

	result, err := client.Explain("asdfnotfound", 127, "zsh: command not found: asdfnotfound")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if result.Explanation == "" {
		t.Error("expected non-empty explanation")
	}

	t.Logf("Got explanation: %s", result.Explanation)
}

func TestExplain_WithFix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateResponse{
			Response: `{"explanation": "Nothing to commit, no staged changes", "fix": "git add ."}`,
			Done:     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		model:      "test-model",
		httpClient: http.DefaultClient,
	}

	result, err := client.Explain("git commit", 1, "nothing to commit, working tree clean")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if result.Fix != "git add ." {
		t.Errorf("expected fix 'git add .', got '%s'", result.Fix)
	}
}

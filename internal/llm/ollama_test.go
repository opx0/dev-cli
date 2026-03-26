package llm

import (
	"dev-cli/internal/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// chatCompletionResponse mirrors the OpenAI chat completion response format.
type chatCompletionResponse struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []chatCompletionChoice  `json:"choices"`
	Usage   chatCompletionUsageResp `json:"usage"`
}

type chatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      chatCompletionMsgResp `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type chatCompletionMsgResp struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionUsageResp struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// newTestOllamaProvider creates an OllamaProvider pointing at a test server.
func newTestOllamaProvider(serverURL string) *OllamaProvider {
	cfg := &config.Config{
		OllamaURL:   serverURL,
		OllamaModel: "test-model",
	}
	return NewOllamaProvider(cfg)
}

// chatCompletionHandler creates a handler that returns the given content.
func chatCompletionHandler(t *testing.T, content string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resp := chatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "test-model",
			Choices: []chatCompletionChoice{
				{
					Index: 0,
					Message: chatCompletionMsgResp{
						Role:    "assistant",
						Content: content,
					},
					FinishReason: "stop",
				},
			},
			Usage: chatCompletionUsageResp{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestExplain_CommandNotFound(t *testing.T) {
	server := httptest.NewServer(chatCompletionHandler(t,
		`{"explanation": "Command 'asdfnotfound' was not found in PATH", "fix": ""}`))
	defer server.Close()

	provider := newTestOllamaProvider(server.URL)
	result, err := provider.Explain("asdfnotfound", 127, "zsh: command not found: asdfnotfound")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if result.Explanation == "" {
		t.Error("expected non-empty explanation")
	}

	t.Logf("Got explanation: %s", result.Explanation)
}

func TestExplain_WithFix(t *testing.T) {
	server := httptest.NewServer(chatCompletionHandler(t,
		`{"explanation": "Nothing to commit, no staged changes", "fix": "git add ."}`))
	defer server.Close()

	provider := newTestOllamaProvider(server.URL)
	result, err := provider.Explain("git commit", 1, "nothing to commit, working tree clean")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if result.Fix != "git add ." {
		t.Errorf("expected fix 'git add .', got '%s'", result.Fix)
	}
}

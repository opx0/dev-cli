package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyzeLog_Parsing(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format (input format check)
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Backend receive invalid JSON input: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		resp := generateResponse{
			Response: `{"explanation": "Root cause is a missing env var", "fix": "export DB_URL=..."}`,
			Done:     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		model:      "test-model",
		httpClient: server.Client(),
	}

	result, err := client.AnalyzeLog("Error: Connection failed")
	if err != nil {
		t.Fatalf("AnalyzeLog failed: %v", err)
	}

	if result.Explanation != "Root cause is a missing env var" {
		t.Errorf("Expected explanation 'Root cause is a missing env var', got '%s'", result.Explanation)
	}
	if result.Fix != "export DB_URL=..." {
		t.Errorf("Expected fix 'export DB_URL=...', got '%s'", result.Fix)
	}
}

func TestAnalyzeLog_MalformedJSON(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateResponse{
			Response: `This is not JSON`,
			Done:     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		model:      "test-model",
		httpClient: server.Client(),
	}

	result, err := client.AnalyzeLog("logs")
	if err != nil {
		t.Fatalf("Should not error on malformed JSON, just return text: %v", err)
	}

	if result.Explanation != "This is not JSON" {
		t.Errorf("Expected raw text as explanation, got %s", result.Explanation)
	}
}

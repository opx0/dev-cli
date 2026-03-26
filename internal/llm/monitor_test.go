package llm

import (
	"net/http/httptest"
	"testing"
)

func TestAnalyzeLog_Parsing(t *testing.T) {
	server := httptest.NewServer(chatCompletionHandler(t,
		`{"explanation": "Root cause is a missing env var", "fix": "export DB_URL=..."}`))
	defer server.Close()

	provider := newTestOllamaProvider(server.URL)
	result, err := provider.AnalyzeLog("Error: Connection failed")
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
	server := httptest.NewServer(chatCompletionHandler(t, `This is not JSON`))
	defer server.Close()

	provider := newTestOllamaProvider(server.URL)
	result, err := provider.AnalyzeLog("logs")
	if err != nil {
		t.Fatalf("Should not error on malformed JSON, just return text: %v", err)
	}

	if result.Explanation != "This is not JSON" {
		t.Errorf("Expected raw text as explanation, got %s", result.Explanation)
	}
}

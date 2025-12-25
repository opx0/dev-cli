package llm

import (
	"os"
	"testing"

	"dev-cli/internal/config"
)

func TestPerplexityConfig(t *testing.T) {

	os.Setenv("DEV_CLI_PERPLEXITY_KEY", "test-key")
	os.Setenv("DEV_CLI_PERPLEXITY_MODEL", "sonar-pro")
	defer os.Unsetenv("DEV_CLI_PERPLEXITY_KEY")
	defer os.Unsetenv("DEV_CLI_PERPLEXITY_MODEL")

	cfg := config.Load()

	if cfg.PerplexityKey != "test-key" {
		t.Errorf("expected PerplexityKey to be 'test-key', got '%s'", cfg.PerplexityKey)
	}
	if cfg.PerplexityModel != "sonar-pro" {
		t.Errorf("expected PerplexityModel to be 'sonar-pro', got '%s'", cfg.PerplexityModel)
	}

	client := NewPerplexityClient(cfg)
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	if client.apiKey != "test-key" {
		t.Errorf("expected client.apiKey to be 'test-key', got '%s'", client.apiKey)
	}
	if client.model != "sonar-pro" {
		t.Errorf("expected client.model to be 'sonar-pro', got '%s'", client.model)
	}
}

func TestPerplexityDefaultConfig(t *testing.T) {
	os.Unsetenv("DEV_CLI_PERPLEXITY_KEY")
	os.Unsetenv("DEV_CLI_PERPLEXITY_MODEL")

	os.Setenv("PERPLEXITY_API_KEY", "legacy-key")
	defer os.Unsetenv("PERPLEXITY_API_KEY")

	cfg := config.Load()

	if cfg.PerplexityModel != "sonar-pro" {
		t.Errorf("expected default PerplexityModel to be 'sonar-pro', got '%s'", cfg.PerplexityModel)
	}

	client := NewPerplexityClient(cfg)
	if client.model != "sonar-pro" {
		t.Errorf("expected client.model to be 'sonar-pro', got '%s'", client.model)
	}
}

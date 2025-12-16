package llm

import (
	"os"
	"testing"

	"dev-cli/internal/config"
)

func TestPerplexityConfig(t *testing.T) {
	// Set up environment variables
	os.Setenv("DEV_CLI_PERPLEXITY_KEY", "test-key")
	os.Setenv("DEV_CLI_PERPLEXITY_MODEL", "sonar-pro")
	defer os.Unsetenv("DEV_CLI_PERPLEXITY_KEY")
	defer os.Unsetenv("DEV_CLI_PERPLEXITY_MODEL")

	// Load config
	cfg := config.Load()

	// Verify config loading
	if cfg.PerplexityKey != "test-key" {
		t.Errorf("expected PerplexityKey to be 'test-key', got '%s'", cfg.PerplexityKey)
	}
	if cfg.PerplexityModel != "sonar-pro" {
		t.Errorf("expected PerplexityModel to be 'sonar-pro', got '%s'", cfg.PerplexityModel)
	}

	// Create client
	client := NewPerplexityClient(cfg)
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	// We can't access private fields directly in test unless we are in the same package (which we are: package llm)
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
	// Start with empty env for these vars
	// but Load() might pick up other env vars if they are set in the system, so we should be careful.
	// However, for defaults check, we just want to see if 'sonar' is default if not set.

	// We need to set key to get a client
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

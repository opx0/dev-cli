package config

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds all settings for the dev-cli application.
// This is the single source of truth — no other config structs should exist.
type Config struct {
	// LLM settings
	OllamaURL       string
	OllamaModel     string
	OllamaUnload    bool
	PerplexityKey   string
	PerplexityModel string
	ForceLocalLLM   bool

	// Storage / logging
	LogDir    string
	LogFormat string

	// Timeouts
	HealthCheckTimeout time.Duration
	LogTimeout         time.Duration
	OperationTimeout   time.Duration
}

// Load creates a Config populated from environment variables with sensible defaults.
func Load() *Config {
	homeDir, _ := os.UserHomeDir()

	cfg := &Config{
		OllamaURL:          "http://localhost:11434",
		OllamaModel:        "qwen2.5-coder:3b-instruct",
		PerplexityModel:    "sonar-pro",
		ForceLocalLLM:      false,
		LogDir:             filepath.Join(homeDir, ".devlogs"),
		LogFormat:          "jsonl",
		HealthCheckTimeout: 5 * time.Second,
		LogTimeout:         10 * time.Second,
		OperationTimeout:   30 * time.Second,
	}

	if val := os.Getenv("DEV_CLI_OLLAMA_URL"); val != "" {
		cfg.OllamaURL = val
	}
	if val := os.Getenv("DEV_CLI_OLLAMA_MODEL"); val != "" {
		cfg.OllamaModel = val
	}
	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		cfg.OllamaUnload = true
	}
	if val := os.Getenv("DEV_CLI_PERPLEXITY_KEY"); val != "" {
		cfg.PerplexityKey = val
	} else if val := os.Getenv("PERPLEXITY_API_KEY"); val != "" {
		cfg.PerplexityKey = val
	}
	if val := os.Getenv("DEV_CLI_PERPLEXITY_MODEL"); val != "" {
		cfg.PerplexityModel = val
	}
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" {
		cfg.ForceLocalLLM = true
	}
	if val := os.Getenv("DEV_CLI_LOG_DIR"); val != "" {
		cfg.LogDir = val
	}
	if val := os.Getenv("DEV_CLI_LOG_FORMAT"); val != "" {
		cfg.LogFormat = val
	}

	return cfg
}

func (c *Config) IsWebSearchEnabled() bool {
	return !c.ForceLocalLLM && c.PerplexityKey != ""
}

var Current = Load()

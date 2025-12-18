package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	OllamaURL       string
	OllamaModel     string
	PerplexityKey   string
	PerplexityModel string
	ForceLocalLLM   bool
	LogDir          string
}

func Load() *Config {
	cfg := &Config{
		OllamaURL:       "http://localhost:11434",
		OllamaModel:     "qwen2.5-coder:3b-instruct",
		PerplexityModel: "sonar-reasoning",
		ForceLocalLLM:   false,
	}

	if val := os.Getenv("DEV_CLI_OLLAMA_URL"); val != "" {
		cfg.OllamaURL = val
	}
	if val := os.Getenv("DEV_CLI_OLLAMA_MODEL"); val != "" {
		cfg.OllamaModel = val
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
	} else {
		home, _ := os.UserHomeDir()
		cfg.LogDir = filepath.Join(home, ".devlogs")
	}

	return cfg
}

func (c *Config) IsWebSearchEnabled() bool {
	return !c.ForceLocalLLM && c.PerplexityKey != ""
}

var Current = Load()

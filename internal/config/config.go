package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
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
	OpenAIURL       string
	OpenAIKey       string
	OpenAIModel     string
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

	// Default log dir: $XDG_DATA_HOME/dev-cli if set, else ~/.devlogs (back-compat).
	defaultLogDir := filepath.Join(homeDir, ".devlogs")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		// Only prefer XDG when ~/.devlogs doesn't already exist, so upgrades don't orphan history.
		legacy := filepath.Join(homeDir, ".devlogs")
		if _, err := os.Stat(legacy); os.IsNotExist(err) {
			defaultLogDir = filepath.Join(xdg, "dev-cli")
		}
	}

	cfg := &Config{
		OllamaURL:          "http://localhost:11434",
		OllamaModel:        "smallthinker",
		PerplexityModel:    "sonar-pro",
		OpenAIURL:          "https://api.openai.com/v1/",
		OpenAIModel:        "gpt-4o-mini",
		ForceLocalLLM:      false,
		LogDir:             defaultLogDir,
		LogFormat:          "jsonl",
		HealthCheckTimeout: 5 * time.Second,
		LogTimeout:         10 * time.Second,
		OperationTimeout:   30 * time.Second,
	}

	// Apply YAML config file (after defaults, before env). Ignore errors —
	// a missing or malformed file should not stop the binary from running.
	applyFile(cfg)

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
	if val := os.Getenv("DEV_CLI_OPENAI_URL"); val != "" {
		cfg.OpenAIURL = val
	} else if val := os.Getenv("OPENAI_BASE_URL"); val != "" {
		cfg.OpenAIURL = val
	}
	if val := os.Getenv("DEV_CLI_OPENAI_KEY"); val != "" {
		cfg.OpenAIKey = val
	} else if val := os.Getenv("OPENAI_API_KEY"); val != "" {
		cfg.OpenAIKey = val
	}
	if val := os.Getenv("DEV_CLI_OPENAI_MODEL"); val != "" {
		cfg.OpenAIModel = val
	} else if val := os.Getenv("OPENAI_MODEL"); val != "" {
		cfg.OpenAIModel = val
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

// HasCloudLLM reports whether any cloud LLM provider is configured (OpenAI-compatible or Perplexity).
func (c *Config) HasCloudLLM() bool {
	return c.OpenAIKey != "" || c.PerplexityKey != ""
}

var Current = Load()

// ── YAML file support ───────────────────────────────────────────────────────
//
// The file schema is intentionally separate from the runtime Config struct so
// we can version / rename keys without breaking callers.

type FileSchema struct {
	Ollama struct {
		URL    string `yaml:"url,omitempty"`
		Model  string `yaml:"model,omitempty"`
		Unload bool   `yaml:"unload,omitempty"`
	} `yaml:"ollama,omitempty"`
	OpenAI struct {
		APIKey  string `yaml:"api_key,omitempty"`
		BaseURL string `yaml:"base_url,omitempty"`
		Model   string `yaml:"model,omitempty"`
	} `yaml:"openai,omitempty"`
	Perplexity struct {
		APIKey string `yaml:"api_key,omitempty"`
		Model  string `yaml:"model,omitempty"`
	} `yaml:"perplexity,omitempty"`
	LogDir    string `yaml:"log_dir,omitempty"`
	LogFormat string `yaml:"log_format,omitempty"`
	ForceLocal bool  `yaml:"force_local,omitempty"`
}

// Path returns the path of the YAML config file: $DEV_CLI_CONFIG if set,
// else <LogDir>/config.yaml. Defaults to ~/.devlogs/config.yaml.
func Path() string {
	if p := os.Getenv("DEV_CLI_CONFIG"); p != "" {
		return p
	}
	homeDir, _ := os.UserHomeDir()
	dir := filepath.Join(homeDir, ".devlogs")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			dir = filepath.Join(xdg, "dev-cli")
		}
	}
	return filepath.Join(dir, "config.yaml")
}

// applyFile loads the YAML config file (if any) and applies its non-empty
// fields to cfg. Env vars (applied later) take precedence.
func applyFile(cfg *Config) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var f FileSchema
	if err := yaml.Unmarshal(data, &f); err != nil {
		return
	}
	if f.Ollama.URL != "" {
		cfg.OllamaURL = f.Ollama.URL
	}
	if f.Ollama.Model != "" {
		cfg.OllamaModel = f.Ollama.Model
	}
	if f.Ollama.Unload {
		cfg.OllamaUnload = true
	}
	if f.OpenAI.APIKey != "" {
		cfg.OpenAIKey = f.OpenAI.APIKey
	}
	if f.OpenAI.BaseURL != "" {
		cfg.OpenAIURL = f.OpenAI.BaseURL
	}
	if f.OpenAI.Model != "" {
		cfg.OpenAIModel = f.OpenAI.Model
	}
	if f.Perplexity.APIKey != "" {
		cfg.PerplexityKey = f.Perplexity.APIKey
	}
	if f.Perplexity.Model != "" {
		cfg.PerplexityModel = f.Perplexity.Model
	}
	if f.LogDir != "" {
		cfg.LogDir = f.LogDir
	}
	if f.LogFormat != "" {
		cfg.LogFormat = f.LogFormat
	}
	if f.ForceLocal {
		cfg.ForceLocalLLM = true
	}
}

// ReadFile returns the file schema as currently on disk (zero value if missing).
// Returns (schema, true) if the file exists, (schema, false) otherwise.
func ReadFile() (FileSchema, bool) {
	data, err := os.ReadFile(Path())
	if err != nil {
		return FileSchema{}, false
	}
	var f FileSchema
	if err := yaml.Unmarshal(data, &f); err != nil {
		return FileSchema{}, false
	}
	return f, true
}

// WriteFile serialises f to the config path, creating parent directories as needed.
func WriteFile(f FileSchema) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(&f)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// SetKey mutates the in-memory file schema by dotted key. Returns an error for
// unknown keys. Supported keys:
//   ollama.url, ollama.model, ollama.unload
//   openai.api_key, openai.base_url, openai.model
//   perplexity.api_key, perplexity.model
//   log_dir, log_format, force_local
func (f *FileSchema) SetKey(key, value string) error {
	switch key {
	case "ollama.url":
		f.Ollama.URL = value
	case "ollama.model":
		f.Ollama.Model = value
	case "ollama.unload":
		f.Ollama.Unload = value == "true" || value == "1" || value == "yes"
	case "openai.api_key":
		f.OpenAI.APIKey = value
	case "openai.base_url":
		f.OpenAI.BaseURL = value
	case "openai.model":
		f.OpenAI.Model = value
	case "perplexity.api_key":
		f.Perplexity.APIKey = value
	case "perplexity.model":
		f.Perplexity.Model = value
	case "log_dir":
		f.LogDir = value
	case "log_format":
		f.LogFormat = value
	case "force_local":
		f.ForceLocal = value == "true" || value == "1" || value == "yes"
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

// GetKey returns the string representation of a dotted key.
func (f FileSchema) GetKey(key string) (string, error) {
	switch key {
	case "ollama.url":
		return f.Ollama.URL, nil
	case "ollama.model":
		return f.Ollama.Model, nil
	case "ollama.unload":
		return fmt.Sprintf("%t", f.Ollama.Unload), nil
	case "openai.api_key":
		return f.OpenAI.APIKey, nil
	case "openai.base_url":
		return f.OpenAI.BaseURL, nil
	case "openai.model":
		return f.OpenAI.Model, nil
	case "perplexity.api_key":
		return f.Perplexity.APIKey, nil
	case "perplexity.model":
		return f.Perplexity.Model, nil
	case "log_dir":
		return f.LogDir, nil
	case "log_format":
		return f.LogFormat, nil
	case "force_local":
		return fmt.Sprintf("%t", f.ForceLocal), nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

// MaskKey returns a masked version of a value for display (e.g. `sk-…abcd`).
// Empty / very short values pass through unchanged.
func MaskKey(v string) string {
	if len(v) <= 8 {
		return v
	}
	return v[:3] + "…" + v[len(v)-4:]
}

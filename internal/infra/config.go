package infra

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	HealthCheckTimeout time.Duration `yaml:"health_check_timeout"`
	LogTimeout         time.Duration `yaml:"log_timeout"`
	OperationTimeout   time.Duration `yaml:"operation_timeout"`
	OllamaBaseURL      string        `yaml:"ollama_base_url"`
	OllamaDefaultModel string        `yaml:"ollama_default_model"`
	DevlogsDir         string        `yaml:"devlogs_dir"`
	LogFormat          string        `yaml:"log_format"`
	OpenCodeCmd        string        `yaml:"opencode_cmd"`
}

func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	devlogsDir := filepath.Join(homeDir, ".devlogs")

	return Config{
		HealthCheckTimeout: 5 * time.Second,
		LogTimeout:         10 * time.Second,
		OperationTimeout:   30 * time.Second,
		OllamaBaseURL:      "http://localhost:11434",
		OllamaDefaultModel: "qwen2.5-coder:3b-instruct",
		DevlogsDir:         devlogsDir,
		LogFormat:          "jsonl",
		OpenCodeCmd:        "opencode",
	}
}

func (c Config) WithHealthCheckTimeout(d time.Duration) Config {
	c.HealthCheckTimeout = d
	return c
}

func (c Config) WithLogTimeout(d time.Duration) Config {
	c.LogTimeout = d
	return c
}

func (c Config) WithOperationTimeout(d time.Duration) Config {
	c.OperationTimeout = d
	return c
}

func (c Config) WithOllamaBaseURL(url string) Config {
	c.OllamaBaseURL = url
	return c
}

func (c Config) WithOllamaDefaultModel(model string) Config {
	c.OllamaDefaultModel = model
	return c
}

func (c Config) WithDevlogsDir(dir string) Config {
	c.DevlogsDir = dir
	return c
}

func (c Config) WithLogFormat(format string) Config {
	c.LogFormat = format
	return c
}

func (c Config) WithOpenCodeCmd(cmd string) Config {
	c.OpenCodeCmd = cmd
	return c
}

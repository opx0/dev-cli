package llm

import (
	"context"
	"os"
	"strings"
)

var webKeywords = []string{
	"install",
	"latest",
	"version",
	"how to",
	"compare",
	"why",
	"best",
	"setup",
	"configure",
	"deploy",
	"update",
	"upgrade",
}

type HybridClient struct {
	perplexity *PerplexityClient
	ollama     *Client
}

func NewHybridClient() *HybridClient {
	return &HybridClient{
		perplexity: NewPerplexityClient(),
		ollama:     NewClient(),
	}
}

func (h *HybridClient) Research(query string) (*ResearchResult, error) {
	if h.perplexity != nil && needsWebSearch(query) {
		result, err := h.perplexity.Research(context.Background(), query)
		if err == nil {
			return result, nil
		}
	}

	return h.ollama.Research(query)
}

func (h *HybridClient) HasPerplexity() bool {
	return h.perplexity != nil
}

func needsWebSearch(query string) bool {
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" {
		return false
	}

	lower := strings.ToLower(query)
	for _, kw := range webKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

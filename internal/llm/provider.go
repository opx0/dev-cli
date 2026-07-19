// Package llm provides a unified interface for LLM providers (Ollama, Perplexity, etc.)
// backed by the OpenAI-compatible chat completions API.
package llm

import (
	"context"
	"encoding/json"
)

// ── Provider Interface ───────────────────────────────────────────────────────

// Provider is the unified interface that all LLM backends must implement.
// Both OllamaProvider and PerplexityProvider satisfy this interface.
type Provider interface {
	// ChatCompletion sends a chat request and returns a response.
	// This is the single entry point for all LLM interactions.
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Name returns the provider identifier (e.g. "ollama", "perplexity").
	Name() string
}

// ── Request / Response Types ─────────────────────────────────────────────────

// ChatRequest represents a chat completion request sent to any provider.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []ToolDef `json:"tools,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int64    `json:"max_tokens,omitempty"`
	Format      string    `json:"format,omitempty"`     // "json" for structured output
	KeepAlive   string    `json:"keep_alive,omitempty"` // Ollama-specific: model unload timer
}

// ChatResponse represents the response from any provider.
type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"` // "stop", "tool_calls", "length"
	Usage        Usage      `json:"usage"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role       string     `json:"role"` // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Only for assistant messages
	ToolCallID string     `json:"tool_call_id,omitempty"` // Only for tool result messages
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // Raw JSON string
}

// ParseArguments unmarshals the tool call arguments into the given target.
func (tc ToolCall) ParseArguments(target any) error {
	return json.Unmarshal([]byte(tc.Arguments), target)
}

// ParseArgumentsMap unmarshals the tool call arguments into a map.
func (tc ToolCall) ParseArgumentsMap() (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(tc.Arguments), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ToolDef defines a tool that the model can call.
// This is the provider-side representation sent in the API request.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// Usage tracks token consumption for a single request.
type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// ── Convenience Constructors ─────────────────────────────────────────────────

// SystemMessage creates a system message.
func SystemMsg(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserMessage creates a user message.
func UserMsg(content string) Message {
	return Message{Role: "user", Content: content}
}

// AssistantMessage creates an assistant message with optional tool calls.
func AssistantMsg(content string, toolCalls ...ToolCall) Message {
	return Message{Role: "assistant", Content: content, ToolCalls: toolCalls}
}

// ToolResultMessage creates a tool result message.
func ToolResultMsg(toolCallID string, content string) Message {
	return Message{Role: "tool", Content: content, ToolCallID: toolCallID}
}

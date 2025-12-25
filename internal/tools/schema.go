// Package tools provides a unified tool abstraction for the RCA agent.
// This file contains JSON Schema generation for LLM tool integration.
package tools

import (
	"encoding/json"
)

// ToolSchema represents a tool in JSON Schema format for LLM integration.
type ToolSchema struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Parameters  ToolSchemaParams `json:"parameters"`
}

// ToolSchemaParams defines the parameters schema for a tool.
type ToolSchemaParams struct {
	Type       string                        `json:"type"`
	Properties map[string]ToolSchemaProperty `json:"properties"`
	Required   []string                      `json:"required"`
}

// ToolSchemaProperty defines a single parameter property.
type ToolSchemaProperty struct {
	Type        string           `json:"type"`
	Description string           `json:"description"`
	Default     any              `json:"default,omitempty"`
	Items       *ToolSchemaItems `json:"items,omitempty"`
}

// ToolSchemaItems defines array item schema.
type ToolSchemaItems struct {
	Type string `json:"type"`
}

// GenerateToolSchema converts a Tool to JSON Schema format.
func GenerateToolSchema(tool Tool) ToolSchema {
	params := tool.Parameters()
	properties := make(map[string]ToolSchemaProperty)
	required := make([]string, 0)

	for _, p := range params {
		prop := ToolSchemaProperty{
			Type:        mapTypeToJSONSchema(p.Type),
			Description: p.Description,
		}

		if p.Type == "[]string" {
			prop.Items = &ToolSchemaItems{Type: "string"}
		} else if p.Type == "[]int" {
			prop.Items = &ToolSchemaItems{Type: "integer"}
		}

		if p.Default != nil {
			prop.Default = p.Default
		}

		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	return ToolSchema{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters: ToolSchemaParams{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// GenerateToolsSchema converts multiple tools to JSON Schema format.
func GenerateToolsSchema(tools []Tool) []ToolSchema {
	schemas := make([]ToolSchema, len(tools))
	for i, tool := range tools {
		schemas[i] = GenerateToolSchema(tool)
	}
	return schemas
}

// mapTypeToJSONSchema converts internal type names to JSON Schema types.
func mapTypeToJSONSchema(internalType string) string {
	switch internalType {
	case "string", "duration":
		return "string"
	case "int":
		return "integer"
	case "bool":
		return "boolean"
	case "[]string", "[]int":
		return "array"
	default:
		return "string"
	}
}

// ToolCallRequest represents a tool call from the LLM.
type ToolCallRequest struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
}

// ToolsPromptJSON returns a JSON string of all tool schemas for LLM prompts.
func ToolsPromptJSON(tools []Tool) (string, error) {
	schemas := GenerateToolsSchema(tools)
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseToolCall parses an LLM response into a ToolCallRequest.
func ParseToolCall(response string) (*ToolCallRequest, error) {
	var call ToolCallRequest
	if err := json.Unmarshal([]byte(response), &call); err != nil {
		return nil, err
	}
	return &call, nil
}

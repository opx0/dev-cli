package tools

// ProviderToolDef is the shared type used to pass tool definitions from the tools
// package to the llm package without creating a circular dependency.
//
// The llm package imports tools, so this type lives here (in tools) and is
// consumed by llm providers when converting registry tools to API-level tool defs.
type ProviderToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// RegistryToProviderToolDefs converts all tools in a Registry to ProviderToolDef
// slice, suitable for passing to an LLM provider.
func RegistryToProviderToolDefs(r *Registry) []ProviderToolDef {
	schemas := r.GetSchemas()
	defs := make([]ProviderToolDef, len(schemas))
	for i, s := range schemas {
		// Convert ToolSchemaParams to a generic map for the JSON Schema
		params := map[string]any{
			"type":       s.Parameters.Type,
			"properties": s.Parameters.Properties,
			"required":   s.Parameters.Required,
		}
		defs[i] = ProviderToolDef{
			Name:        s.Name,
			Description: s.Description,
			Parameters:  params,
		}
	}
	return defs
}

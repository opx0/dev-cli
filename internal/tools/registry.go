package tools

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages tool registration and lookup.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// NewDefaultRegistry returns a fresh registry containing the standard RCA tools.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.RegisterDefaults()
	return r
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// MustRegister adds a tool to the registry, panicking on error.
func (r *Registry) MustRegister(tool Tool) {
	if err := r.Register(tool); err != nil {
		panic(err)
	}
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns information about all registered tools.
func (r *Registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		infos = append(infos, ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// Names returns the names of all registered tools.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// RegisterDefaults registers all default tools.
// Call this to populate the registry with standard RCA tools.
func (r *Registry) RegisterDefaults() {
	r.MustRegister(&ReadFileTool{})
	r.MustRegister(&ReadDirTool{})
	r.MustRegister(&WriteFileTool{})
	r.MustRegister(&RunCommandTool{})
	r.MustRegister(&SearchCodebaseTool{})
	r.MustRegister(&QueryDockerTool{})
	r.MustRegister(&CheckPortsTool{})
	r.MustRegister(&GitInfoTool{})
	r.MustRegister(&PackageInfoTool{})
	r.MustRegister(&ReadDiffTool{})
}

// GetSchemas returns JSON schemas for all registered tools.
func (r *Registry) GetSchemas() []ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return GenerateToolsSchema(tools)
}

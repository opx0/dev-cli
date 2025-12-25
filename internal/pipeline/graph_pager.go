package pipeline

import (
	"context"
	"time"
)

// CausalNode represents a node in the causal dependency graph.
type CausalNode struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "error", "service", "dependency", "config"
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Level       int               `json:"level"` // Depth in causal chain (0 = root cause)
	Metadata    map[string]string `json:"metadata,omitempty"`
	Children    []string          `json:"children,omitempty"` // Child node IDs
}

// GraphPager handles paginated traversal of causal graphs.
// Useful for large dependency graphs during RCA.
type GraphPager struct {
	PageSize int
	MaxDepth int
	nodes    map[string]*CausalNode
}

// NewGraphPager creates a pager with configurable page size and max depth.
func NewGraphPager(pageSize, maxDepth int) *GraphPager {
	if pageSize <= 0 {
		pageSize = 20
	}
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &GraphPager{
		PageSize: pageSize,
		MaxDepth: maxDepth,
		nodes:    make(map[string]*CausalNode),
	}
}

// Page represents a subset of graph nodes for analysis.
type Page struct {
	Nodes      []CausalNode `json:"nodes"`
	TotalNodes int          `json:"total_nodes"`
	HasMore    bool         `json:"has_more"`
	NextCursor string       `json:"next_cursor,omitempty"`
	Level      int          `json:"level"`
}

// AddNode adds a node to the graph.
func (p *GraphPager) AddNode(node CausalNode) {
	p.nodes[node.ID] = &node
}

// AddNodes adds multiple nodes to the graph.
func (p *GraphPager) AddNodes(nodes []CausalNode) {
	for _, node := range nodes {
		p.AddNode(node)
	}
}

// GetNode retrieves a node by ID.
func (p *GraphPager) GetNode(id string) *CausalNode {
	return p.nodes[id]
}

// NodeCount returns the total number of nodes.
func (p *GraphPager) NodeCount() int {
	return len(p.nodes)
}

// Clear removes all nodes from the graph.
func (p *GraphPager) Clear() {
	p.nodes = make(map[string]*CausalNode)
}

// GetPage returns a page of nodes at or below a specific level.
func (p *GraphPager) GetPage(ctx context.Context, level int, cursor string) (*Page, error) {

	nodesAtLevel := make([]CausalNode, 0)
	for _, node := range p.nodes {
		if node.Level == level {
			nodesAtLevel = append(nodesAtLevel, *node)
		}
	}

	startIdx := 0
	if cursor != "" {
		for i, n := range nodesAtLevel {
			if n.ID == cursor {
				startIdx = i
				break
			}
		}
	}

	endIdx := startIdx + p.PageSize
	if endIdx > len(nodesAtLevel) {
		endIdx = len(nodesAtLevel)
	}

	pageNodes := nodesAtLevel[startIdx:endIdx]

	page := &Page{
		Nodes:      pageNodes,
		TotalNodes: len(nodesAtLevel),
		HasMore:    endIdx < len(nodesAtLevel),
		Level:      level,
	}

	if page.HasMore && len(pageNodes) > 0 {
		page.NextCursor = pageNodes[len(pageNodes)-1].ID
	}

	return page, nil
}

// GetNodesAtLevel returns all nodes at a specific depth level.
func (p *GraphPager) GetNodesAtLevel(level int) []CausalNode {
	result := make([]CausalNode, 0)
	for _, node := range p.nodes {
		if node.Level == level {
			result = append(result, *node)
		}
	}
	return result
}

// GetRootCauses returns all level-0 (root cause) nodes.
func (p *GraphPager) GetRootCauses() []CausalNode {
	return p.GetNodesAtLevel(0)
}

// GetChildren returns all direct children of a node.
func (p *GraphPager) GetChildren(nodeID string) []CausalNode {
	parent := p.nodes[nodeID]
	if parent == nil {
		return nil
	}

	children := make([]CausalNode, 0, len(parent.Children))
	for _, childID := range parent.Children {
		if child := p.nodes[childID]; child != nil {
			children = append(children, *child)
		}
	}
	return children
}

// TraverseBFS performs breadth-first traversal from a starting node.
// Calls the visitor function for each node, stopping if it returns false.
func (p *GraphPager) TraverseBFS(ctx context.Context, startID string, visitor func(CausalNode) bool) error {
	visited := make(map[string]bool)
	queue := []string{startID}

	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nodeID := queue[0]
		queue = queue[1:]

		if visited[nodeID] {
			continue
		}
		visited[nodeID] = true

		node := p.nodes[nodeID]
		if node == nil {
			continue
		}

		if !visitor(*node) {
			return nil
		}

		queue = append(queue, node.Children...)
	}

	return nil
}

// BuildFromFailure creates a causal graph from a failure block.
// This is a starting point - actual causal analysis would involve LLM.
func (p *GraphPager) BuildFromFailure(failure *Block) {

	root := CausalNode{
		ID:          failure.ID,
		Type:        "error",
		Name:        failure.Command,
		Description: truncateString(failure.Output, 200),
		Level:       0,
		Metadata: map[string]string{
			"exit_code":   string(rune('0' + failure.ExitCode)),
			"working_dir": failure.WorkingDir,
			"timestamp":   failure.Timestamp.Format(time.RFC3339),
		},
	}
	p.AddNode(root)
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Package mcp provides an MCP server implementation for dev-cli.
// This exposes debugging tools to OpenCode and other MCP clients.
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewDevCLIMCPServer creates a new MCP server with all dev-cli debugging tools.
func NewDevCLIMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"dev-cli-debugger",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	// Register all debugging tools
	registerQueryHistoryTool(s)
	registerFindSimilarFailuresTool(s)
	registerGetSolutionsTool(s)
	registerStoreSolutionTool(s)
	registerGetProjectFingerprintTool(s)

	return s
}

// Serve starts the MCP server using stdio transport.
func Serve() error {
	s := NewDevCLIMCPServer()
	return server.ServeStdio(s)
}

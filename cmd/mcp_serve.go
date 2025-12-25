package cmd

import (
	"fmt"
	"os"

	"dev-cli/internal/mcp"

	"github.com/spf13/cobra"
)

var mcpServeCmd = &cobra.Command{
	Use:   "mcp-serve",
	Short: "Start MCP server for OpenCode integration",
	Long: `Start a Model Context Protocol (MCP) server that exposes dev-cli's
debugging tools to OpenCode and other MCP-compatible AI agents.

The server provides tools for:
  - Querying command history
  - Finding similar failures
  - Getting and storing solutions
  - Project fingerprinting

To use with OpenCode, add this to your opencode.json:
  {
    "mcp": {
      "dev-cli": {
        "type": "local",
        "command": ["dev-cli", "mcp-serve"],
        "enabled": true
      }
    }
  }`,
	Example: `  # Start MCP server (for use with OpenCode)
  dev-cli mcp-serve

  # Test MCP server manually (sends JSON-RPC via stdin)
  echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | dev-cli mcp-serve`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcp.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpServeCmd)
}

package cmd

import (
	"fmt"
	"os"

	"dev-cli/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive dashboard",
	Long: `Launch the Terminal User Interface (TUI) - your Mission Control.

Tabs:
  Dashboard  - System status and metrics
  Monitor    - Real-time log watching with AI
  Assist     - Chat interface for asking questions
  History    - Searchable command history

Navigation: Use Tab/Shift+Tab or number keys. Press 'q' to quit.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running dashboard: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

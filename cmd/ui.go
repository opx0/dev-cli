package cmd

import (
	"fmt"

	"dev-cli/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive dashboard",
	Long: `Launch the focused Terminal User Interface (TUI).

Tabs:
  Containers - Container status and recent logs
  History    - Recent command failures

Navigation: Use Tab/Shift+Tab or number keys. Press 'q' to quit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("run dashboard: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

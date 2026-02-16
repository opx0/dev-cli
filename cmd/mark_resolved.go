package cmd

import (
	"fmt"
	"os"

	"dev-cli/internal/storage"

	"github.com/spf13/cobra"
)

var (
	resolveID         int64
	resolveResolution string
)

var markResolvedCmd = &cobra.Command{
	Use:    "mark-resolved",
	Short:  "Mark a failed command as resolved",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if resolveID <= 0 {
			fmt.Fprintln(os.Stderr, "error: --id is required")
			os.Exit(1)
		}

		validResolutions := map[string]bool{
			"solution":  true,
			"unrelated": true,
			"skipped":   true,
		}
		if !validResolutions[resolveResolution] {
			fmt.Fprintf(os.Stderr, "error: --resolution must be one of: solution, unrelated, skipped\n")
			os.Exit(1)
		}

		db := storage.DB()

		if err := storage.MarkResolution(db, resolveID, resolveResolution); err != nil {
			printError(fmt.Sprintf("marking resolution: %v", err))
			os.Exit(1)
		}
	},
}

var checkLastFailureCmd = &cobra.Command{
	Use:    "check-last-failure",
	Short:  "Check if there's an unresolved failure",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		db := storage.DB()

		failure, err := storage.GetLastUnresolvedFailure(db)
		if err != nil {
			os.Exit(1)
		}
		if failure == nil {
			os.Exit(1)
		}

		cmdStr := failure.Command
		if len(cmdStr) > 50 {
			cmdStr = cmdStr[:47] + "..."
		}
		fmt.Printf("%d|%s\n", failure.ID, cmdStr)
	},
}

func init() {
	rootCmd.AddCommand(markResolvedCmd)
	markResolvedCmd.Flags().Int64Var(&resolveID, "id", 0, "History entry ID")
	markResolvedCmd.Flags().StringVar(&resolveResolution, "resolution", "", "Resolution type: solution, unrelated, skipped")

	rootCmd.AddCommand(checkLastFailureCmd)
}

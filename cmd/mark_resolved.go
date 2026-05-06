package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dev-cli/internal/config"
	"dev-cli/internal/memory"
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

		if resolveResolution == "solution" {
			storeResolutionMemory(db, resolveID)
		}
	},
}

// storeResolutionMemory persists a `problem` memory describing a failure the
// user confirmed was resolved. We pull the failed command + captured output
// from history. We do not know which subsequent command served as the fix,
// but the failure context alone is useful recall material — the LLM can
// reason about it the next time the same error pattern shows up.
func storeResolutionMemory(db *sql.DB, id int64) {
	cfg := config.Load()
	if !cfg.MemPalaceEnabled || !cfg.MemPalaceWriteback {
		return
	}
	if db == nil {
		return
	}

	item, err := storage.GetHistoryByID(db, id)
	if err != nil || item == nil {
		return
	}

	output := ""
	if item.Details != "" {
		var details map[string]any
		if json.Unmarshal([]byte(item.Details), &details) == nil {
			if s, ok := details["output"].(string); ok {
				output = s
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Command (exit %d): %s\n", item.ExitCode, item.Command)
	if output = strings.TrimSpace(output); output != "" {
		if len(output) > 800 {
			output = output[:800] + "…"
		}
		fmt.Fprintf(&b, "\nOutput:\n%s\n", output)
	}
	fmt.Fprintf(&b, "\nResolved by user (resolution=%s)", resolveResolution)
	text := strings.TrimRight(b.String(), "\n")

	cwd := item.Directory
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = memory.Store(ctx, cfg, memory.StoreReq{
			Hall:   "problem",
			Cwd:    cwd,
			Text:   text,
			Source: fmt.Sprintf("dev-cli/explain:history-%d", id),
		})
	}()
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

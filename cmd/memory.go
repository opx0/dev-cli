package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/config"
	"dev-cli/internal/memory"

	"github.com/spf13/cobra"
)

var (
	memSearchLimit int
	memSearchWing  string
	memSearchHall  string
	memStoreWing   string
	memStoreHall   string
	memStoreSource string
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Hybrid recall over prior dev-cli runs and notes (via MemPalace)",
	Long: `Surfaces and writes long-term memories using the MemPalace CLI.
Recall is hybrid (vector + BM25 + cross-encoder rerank) over Claude Code
transcripts, Obsidian notes, and dev-cli's own outcomes.

Requires MemPalace to be installed (mempalace or mp on PATH) and enabled
via DEV_CLI_MEMPALACE_ENABLED=1 or mempalace.enabled: true in the config.`,
	Example: `  dev-cli memory search "kubectl rollout"
  dev-cli memory search --hall problem "ENOSPC" --limit 8
  dev-cli memory stats
  dev-cli memory ingest
  dev-cli memory store --hall advice "Always pin sqlite-vec to 0.1.x on Arch"`,
}

var memorySearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search prior memories",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runMemorySearch,
}

var memoryStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show MemPalace index counts",
	RunE:  runMemoryStats,
}

var memoryIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Run an incremental MemPalace ingest from Claude Code transcripts",
	RunE:  runMemoryIngest,
}

var memoryStoreCmd = &cobra.Command{
	Use:   "store [text]",
	Short: "Store a memory manually",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runMemoryStore,
}

func init() {
	memorySearchCmd.Flags().IntVarP(&memSearchLimit, "limit", "n", 5, "Number of hits to show")
	memorySearchCmd.Flags().StringVar(&memSearchWing, "wing", "", "Filter by wing (project scope)")
	memorySearchCmd.Flags().StringVar(&memSearchHall, "hall", "", "Filter by hall (fact, decision, problem, advice, snippet, …)")

	memoryStoreCmd.Flags().StringVar(&memStoreWing, "wing", "", "Wing for the new memory (default: auto-derived)")
	memoryStoreCmd.Flags().StringVar(&memStoreHall, "hall", "advice", "Hall for the new memory")
	memoryStoreCmd.Flags().StringVar(&memStoreSource, "source", "dev-cli/manual", "Source tag for the new memory")

	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryStatsCmd)
	memoryCmd.AddCommand(memoryIngestCmd)
	memoryCmd.AddCommand(memoryStoreCmd)
	rootCmd.AddCommand(memoryCmd)
}

func ensureMemoryReady(cfg *config.Config) error {
	if !cfg.MemPalaceEnabled {
		return fmt.Errorf("MemPalace is disabled — set DEV_CLI_MEMPALACE_ENABLED=1 or mempalace.enabled in config")
	}
	if !memory.Available() {
		return fmt.Errorf("mempalace CLI not on PATH (install via: pipx install mempalace, or use the bundled `mp` binary)")
	}
	return nil
}

func runMemorySearch(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	if err := ensureMemoryReady(cfg); err != nil {
		return err
	}
	query := strings.Join(args, " ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hits, err := memory.Search(ctx, cfg, query, memory.SearchOpts{
		Limit: memSearchLimit,
		Wing:  memSearchWing,
		Hall:  memSearchHall,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(hits) == 0 {
		printInfo("No memories found")
		return nil
	}

	fmt.Printf("\n%s%d hit(s) for: %q%s\n\n", colorBold, len(hits), query, colorReset)
	for i, h := range hits {
		fmt.Printf("%s[%d]%s %s%.3f%s  %s%s/%s%s\n",
			colorCyan, i+1, colorReset,
			colorGray, h.Score, colorReset,
			colorYellow, safeStr(h.Wing, "·"), safeStr(h.Hall, "·"), colorReset)
		text := strings.TrimSpace(h.Text)
		const max = 400
		if len(text) > max {
			text = text[:max] + "…"
		}
		for _, line := range strings.Split(text, "\n") {
			fmt.Printf("    %s\n", line)
		}
		if h.Source != "" {
			fmt.Printf("    %ssource: %s%s\n", colorGray, h.Source, colorReset)
		}
		fmt.Println()
	}
	return nil
}

func runMemoryStats(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	if err := ensureMemoryReady(cfg); err != nil {
		return err
	}
	return passthroughMempalace("stats")
}

func runMemoryIngest(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	if err := ensureMemoryReady(cfg); err != nil {
		return err
	}
	return passthroughMempalace("ingest")
}

func runMemoryStore(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	if err := ensureMemoryReady(cfg); err != nil {
		return err
	}
	text := strings.Join(args, " ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := memory.Store(ctx, cfg, memory.StoreReq{
		Hall:   memStoreHall,
		Wing:   memStoreWing,
		Text:   text,
		Source: memStoreSource,
	})
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}
	printSuccess("Memory stored")
	return nil
}

// passthroughMempalace runs the underlying CLI with stdio inherited so the
// user sees its output verbatim. Used for stats and ingest where dev-cli has
// no value-add over a direct invocation.
func passthroughMempalace(subcommand string) error {
	bin, err := exec.LookPath("mempalace")
	if err != nil {
		bin, err = exec.LookPath("mp")
		if err != nil {
			return fmt.Errorf("mempalace CLI not on PATH")
		}
	}
	c := exec.Command(bin, subcommand)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func safeStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

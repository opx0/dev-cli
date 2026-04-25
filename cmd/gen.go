package cmd

import (
	"context"
	"fmt"
	"strings"

	"dev-cli/internal/config"
	"dev-cli/internal/llm"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/tools"
	"dev-cli/internal/workflow"

	"github.com/spf13/cobra"
)

var (
	genMaxIter     int
	genAutoApprove bool
)

var genCmd = &cobra.Command{
	Use:   "gen <kind> <target>",
	Short: "Scaffold code with an AI agent (test/migration/fixture)",
	Long: `Generate common code scaffolds via the same safe-mode agent loop
used by dev-cli fix, but with tools limited to read_file / write_file /
search_codebase / read_dir.

Kinds:
  test <symbol-or-file>     Generate/extend a unit test for a function or file.
  migration <intent>        Draft a database migration that satisfies <intent>.
  fixture <type-or-file>    Build fixture data (struct literal / JSON / SQL) for a type.`,
	Example: `  dev-cli gen test SelectAgentModel
  dev-cli gen migration "add email_verified boolean to users, default false"
  dev-cli gen fixture User`,
	Args: cobra.MinimumNArgs(2),
	RunE: runGen,
}

func init() {
	genCmd.Flags().IntVar(&genMaxIter, "max-iterations", 8, "Maximum agent iterations")
	genCmd.Flags().BoolVar(&genAutoApprove, "auto-approve", false, "Auto-approve file writes (dangerous)")
	rootCmd.AddCommand(genCmd)
}

func runGen(cmd *cobra.Command, args []string) error {
	kind := strings.ToLower(args[0])
	target := strings.Join(args[1:], " ")

	systemPrompt, err := buildGenSystemPrompt(kind, target)
	if err != nil {
		return err
	}

	// Build a restricted registry: only the subset we want the agent to touch.
	full := tools.GetRegistry()
	full.RegisterDefaults()
	restricted := tools.NewRegistry()
	for _, name := range []string{"read_file", "read_dir", "write_file", "search_codebase"} {
		if t, ok := full.Get(name); ok {
			_ = restricted.Register(t)
		}
	}

	cfg := config.Load()
	providerName, model := llm.SelectAgentModel(cfg, false)

	bus := pipeline.NewEventBus()
	engine := workflow.NewEngine(nil, bus)
	if !genAutoApprove {
		engine.SetSafeMode(workflow.NewExecuteContext(func(action string) bool {
			fmt.Printf("\n%s %s\n  Allow? [y/N]: ", iconWarn(), action)
			var resp string
			fmt.Scanln(&resp)
			return resp == "y" || resp == "Y"
		}))
	} else {
		engine.SetSafeMode(workflow.NewExecuteContext(func(string) bool { return true }))
	}

	agent := llm.NewAgentWithDeps(llm.NewHybridClient(), restricted, engine)
	agent.SetConfig(llm.AgentConfig{
		MaxIterations: genMaxIter,
		Model:         model,
		Verbose:       false,
		ApprovalFunc: func(toolName string, params map[string]any) bool {
			if genAutoApprove {
				return true
			}
			if toolName != "write_file" {
				return true
			}
			path, _ := params["path"].(string)
			fmt.Printf("\n%s write_file: %s\n  Allow? [y/N]: ", iconInfo(), path)
			var resp string
			fmt.Scanln(&resp)
			return resp == "y" || resp == "Y"
		},
	})

	fmt.Printf("%s Scaffolding (%s/%s): %s %s\n", iconInfo(), providerName, model, kind, target)

	ctx := context.Background()
	task := fmt.Sprintf("%s: %s", kind, target)
	result, err := agent.Run(ctx, task, systemPrompt)
	if err != nil {
		return fmt.Errorf("agent: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("agent could not finish: %s", result.Error)
	}
	if result.Summary != "" {
		fmt.Println()
		fmt.Println(result.Summary)
	}
	return nil
}

func buildGenSystemPrompt(kind, target string) (string, error) {
	base := `You are a code scaffolding assistant. Tools available: read_file, read_dir, search_codebase, write_file.

Process:
1. Use read_dir / search_codebase to find the right location and conventions in this project.
2. Use read_file to study the style of existing code (imports, naming, assertion library).
3. Call write_file at most ONCE per output file. Match the project's style.
4. When finished, respond WITHOUT calling more tools, with a one-paragraph summary of what was written.

Absolute rules:
- Do not modify unrelated files.
- Do not invent imports or APIs that don't exist in the project — verify first.
- Do not write comments that restate the code.`

	switch kind {
	case "test":
		return base + fmt.Sprintf(`

TASK: write a unit test for: %s

Find where this symbol is defined. If the project already has tests in a neighbouring *_test file, follow that file's structure. Cover the golden path and one or two edge cases — not every permutation.`, target), nil
	case "migration":
		return base + fmt.Sprintf(`

TASK: draft a database migration for this intent: %s

Find the project's migrations folder (likely migrations/, db/migrations/, or similar). Match the existing migration framework (raw SQL, Alembic, Flyway, golang-migrate, etc.). Include both up and down where the framework expects them.`, target), nil
	case "fixture":
		return base + fmt.Sprintf(`

TASK: generate fixture data for: %s

Find the type definition first. Produce a self-contained fixture in the project's conventional location (testdata/, fixtures/, or a _test.go helper). Include 2–3 representative rows/instances.`, target), nil
	default:
		return "", fmt.Errorf("unknown kind %q (valid: test, migration, fixture)", kind)
	}
}

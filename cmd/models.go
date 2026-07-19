package cmd

import (
	"dev-cli/internal/config"
	"dev-cli/internal/infra"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage local LLM models",
	Long:  `List and pull local models used by dev-cli via Ollama.`,
	RunE:  runModelsList,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available local models",
	RunE:  runModelsList,
}

var modelsPullCmd = &cobra.Command{
	Use:   "pull [model]",
	Short: "Pull a model from the Ollama registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsPull,
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsListCmd, modelsPullCmd)
}

func getOllamaClient() *infra.OllamaClient {
	return infra.NewOllamaClient(config.Load().OllamaURL)
}

func runModelsList(cmd *cobra.Command, _ []string) error {
	cfg := config.Load()
	client := infra.NewOllamaClient(cfg.OllamaURL)
	ctx := cmd.Context()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ollama is not running or unreachable at %s; run 'dev-cli doctor'", client.BaseURL())
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	if len(models) == 0 {
		fmt.Println("No models installed.")
		fmt.Println("Use 'dev-cli models pull <model>' to download one.")
		return nil
	}

	fmt.Printf("%s Available Models:\n", iconInfo())
	separator()
	for _, model := range models {
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		isDefault := ""
		if strings.HasPrefix(model.Name, cfg.OllamaModel) {
			isDefault = colorGreen + " (default)" + colorReset
		}
		fmt.Printf("  • %-25s %5.1f GB%s\n", fmtBold(model.Name), sizeGB, isDefault)
	}
	fmt.Println()
	return nil
}

func runModelsPull(cmd *cobra.Command, args []string) error {
	modelName := args[0]
	client := getOllamaClient()
	ctx := cmd.Context()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ollama is not running or unreachable at %s; run 'dev-cli doctor'", client.BaseURL())
	}

	fmt.Printf("%s Pulling model '%s'...\n", iconInfo(), modelName)
	err := client.PullModelSync(ctx, modelName, func(progress infra.OllamaPullProgress) {
		if progress.Total > 0 {
			percent := float64(progress.Completed) / float64(progress.Total) * 100
			fmt.Printf("\r  %s %.1f%% (%d/%d MB)", progress.Status, percent, progress.Completed/(1024*1024), progress.Total/(1024*1024))
			return
		}
		fmt.Printf("\r  %s", progress.Status)
	})
	_, _ = fmt.Fprintln(os.Stdout)

	if err != nil {
		return fmt.Errorf("pull model: %w", err)
	}

	printSuccess(fmt.Sprintf("Successfully pulled model '%s'.", modelName))
	return nil
}

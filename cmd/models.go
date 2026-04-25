package cmd

import (
	"context"
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
	Long:  `List, pull, and manage local models used by dev-cli via Ollama.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to list
		runModelsList(cmd, args)
	},
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available local models",
	Run:   runModelsList,
}

var modelsPullCmd = &cobra.Command{
	Use:   "pull [model]",
	Short: "Pull a model from the Ollama registry",
	Args:  cobra.ExactArgs(1),
	Run:   runModelsPull,
}

var modelsRmCmd = &cobra.Command{
	Use:   "rm [model]",
	Short: "Remove a model from the Ollama registry",
	Args:  cobra.ExactArgs(1),
	Run:   runModelsRm,
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsPullCmd)
	modelsCmd.AddCommand(modelsRmCmd)
}

func getOllamaClient() *infra.OllamaClient {
	// Attempt to get docker client, but continue even if it fails
	dockerClient, _ := infra.NewDockerClient()
	return infra.NewOllamaClient(dockerClient, config.Current.OllamaURL)
}

func runModelsList(cmd *cobra.Command, args []string) {
	client := getOllamaClient()
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		printError(fmt.Sprintf("Ollama is not running or unreachable at %s.", client.BaseURL()))
		printInfo("Run 'dev-cli doctor' or 'make setup' to ensure Ollama is running.")
		os.Exit(1)
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		printError(fmt.Sprintf("Failed to list models: %v", err))
		os.Exit(1)
	}

	if len(models) == 0 {
		fmt.Println("No models installed.")
		fmt.Println("Use 'dev-cli models pull <model>' to download one.")
		return
	}

	fmt.Printf("%s Available Models:\n", iconInfo())
	separator()
	
	for _, m := range models {
		sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
		isDefault := ""
		if strings.HasPrefix(m.Name, config.Current.OllamaModel) {
			isDefault = colorGreen + " (default)" + colorReset
		}
		fmt.Printf("  • %-25s %5.1f GB%s\n", fmtBold(m.Name), sizeGB, isDefault)
	}
	fmt.Println()
}

func runModelsPull(cmd *cobra.Command, args []string) {
	modelName := args[0]
	client := getOllamaClient()
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		printError(fmt.Sprintf("Ollama is not running or unreachable at %s.", client.BaseURL()))
		printInfo("Run 'dev-cli doctor' or 'make setup' to ensure Ollama is running.")
		os.Exit(1)
	}

	fmt.Printf("%s Pulling model '%s'...\n", iconInfo(), modelName)
	
	err := client.PullModelSync(ctx, modelName, func(progress infra.OllamaPullProgress) {
		if progress.Total > 0 {
			percent := float64(progress.Completed) / float64(progress.Total) * 100
			fmt.Printf("\r  %s %.1f%% (%d/%d MB)", progress.Status, percent, progress.Completed/(1024*1024), progress.Total/(1024*1024))
		} else {
			fmt.Printf("\r  %s", progress.Status)
		}
	})

	fmt.Println() // Newline after progress

	if err != nil {
		printError(fmt.Sprintf("Failed to pull model: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Successfully pulled model '%s'.", modelName))
}

func runModelsRm(cmd *cobra.Command, args []string) {
	modelName := args[0]
	client := getOllamaClient()
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		printError(fmt.Sprintf("Ollama is not running or unreachable at %s.", client.BaseURL()))
		printInfo("Run 'dev-cli doctor' or 'make setup' to ensure Ollama is running.")
		os.Exit(1)
	}

	fmt.Printf("%s Removing model '%s'...\n", iconInfo(), modelName)

	if err := client.DeleteModel(ctx, modelName); err != nil {
		printError(fmt.Sprintf("Failed to remove model: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Successfully removed model '%s'.", modelName))
}

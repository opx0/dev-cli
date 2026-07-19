package cmd

import (
	"fmt"
	"os"
	"strings"

	"dev-cli/internal/config"

	"github.com/spf13/cobra"
)

var configForce bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dev-cli configuration file (YAML)",
	Long: `Manage the dev-cli YAML configuration.

The config file is read at $DEV_CLI_CONFIG if set, else <LogDir>/config.yaml
(default: ~/.devlogs/config.yaml). Values are applied with precedence:

    env vars > config file > defaults

Available keys:
  ollama.url, ollama.model
  openai.api_key, openai.base_url, openai.model
  perplexity.api_key, perplexity.model
  log_dir, force_local`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file at the standard location",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.Path()
		if _, err := os.Stat(path); err == nil && !configForce {
			return fmt.Errorf("%s already exists; re-run with --force to overwrite", path)
		}

		var f config.FileSchema
		f.Ollama.URL = "http://localhost:11434"
		f.Ollama.Model = "smallthinker"
		f.OpenAI.BaseURL = "https://api.openai.com/v1/"
		f.OpenAI.Model = "gpt-4o-mini"
		f.Perplexity.Model = "sonar-pro"

		if err := config.WriteFile(f); err != nil {
			return err
		}
		fmt.Printf("%s Wrote default config to %s\n", iconOK(), path)
		fmt.Printf("%s Edit the file or set credentials through environment variables.\n", iconInfo())
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective config (file values; API keys masked)",
	RunE: func(cmd *cobra.Command, args []string) error {
		f, ok := config.ReadFile()
		if !ok {
			fmt.Printf("%s No config file at %s (using defaults + env)\n", iconWarn(), config.Path())
			return nil
		}

		fmt.Printf("Config file: %s\n\n", config.Path())
		fmt.Println("ollama:")
		fmt.Printf("  url:        %s\n", f.Ollama.URL)
		fmt.Printf("  model:      %s\n", f.Ollama.Model)
		fmt.Println("openai:")
		fmt.Printf("  api_key:    %s\n", config.MaskKey(f.OpenAI.APIKey))
		fmt.Printf("  base_url:   %s\n", f.OpenAI.BaseURL)
		fmt.Printf("  model:      %s\n", f.OpenAI.Model)
		fmt.Println("perplexity:")
		fmt.Printf("  api_key:    %s\n", config.MaskKey(f.Perplexity.APIKey))
		fmt.Printf("  model:      %s\n", f.Perplexity.Model)
		fmt.Printf("log_dir:     %s\n", f.LogDir)
		fmt.Printf("force_local: %t\n", f.ForceLocal)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config key (e.g. dev-cli config set openai.api_key sk-…)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.TrimSpace(args[0])
		value := args[1]
		if strings.HasSuffix(key, "api_key") {
			return fmt.Errorf("refusing API key on the command line; edit %s or use an environment variable", config.Path())
		}

		f, _ := config.ReadFile()
		if err := f.SetKey(key, value); err != nil {
			return err
		}
		if err := config.WriteFile(f); err != nil {
			return err
		}
		displayVal := value
		if strings.HasSuffix(key, "api_key") {
			displayVal = config.MaskKey(value)
		}
		fmt.Printf("%s %s = %s\n", iconOK(), key, displayVal)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, ok := config.ReadFile()
		if !ok {
			return fmt.Errorf("no config file at %s", config.Path())
		}
		val, err := f.GetKey(args[0])
		if err != nil {
			return err
		}
		if strings.HasSuffix(args[0], "api_key") {
			val = config.MaskKey(val)
		}
		fmt.Println(val)
		return nil
	},
}

func init() {
	configInitCmd.Flags().BoolVar(&configForce, "force", false, "Overwrite existing config file")
	configCmd.AddCommand(configInitCmd, configShowCmd, configSetCmd, configGetCmd)
	rootCmd.AddCommand(configCmd)
}

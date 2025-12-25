package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	exportDocker string
	exportFile   string
	exportLines  int
	exportSave   bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export logs for OpenCode ingestion",
	Long: `Export container or file logs in a format suitable for OpenCode.
Output can be piped or saved to ~/.devlogs/last-error.md for use with OpenCode.`,
	Example: `  # Export Docker container logs
  dev-cli export --docker my-container --lines 50

  # Export and save for OpenCode handoff
  dev-cli export --docker my-container --save

  # Export from a log file
  dev-cli export --file /var/log/app.log --lines 100`,
	Run: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVar(&exportDocker, "docker", "", "Docker container ID/name to export logs from")
	exportCmd.Flags().StringVar(&exportFile, "file", "", "Log file path to export from")
	exportCmd.Flags().IntVar(&exportLines, "lines", 50, "Number of log lines to export")
	exportCmd.Flags().BoolVar(&exportSave, "save", false, "Save to ~/.devlogs/last-error.md for OpenCode handoff")
}

func runExport(cmd *cobra.Command, args []string) {
	if exportDocker == "" && exportFile == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify --docker or --file")
		cmd.Usage()
		os.Exit(1)
	}

	var logs string
	var source string
	var err error

	if exportDocker != "" {
		source = fmt.Sprintf("Docker container: %s", exportDocker)
		logs, err = getDockerLogs(exportDocker, exportLines)
	} else {
		source = fmt.Sprintf("Log file: %s", exportFile)
		logs, err = getFileLogs(exportFile, exportLines)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output := formatForOpenCode(source, logs)

	if exportSave {
		savePath, err := saveForOpenCode(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\033[32mSaved to: %s\033[0m\n", savePath)
		fmt.Fprintf(os.Stderr, "\033[36mRun 'opencode' and use: @%s\033[0m\n", savePath)
	} else {
		fmt.Print(output)
	}
}

func getDockerLogs(container string, lines int) (string, error) {
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", lines), container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w\n%s", err, string(output))
	}
	return string(output), nil
}

func getFileLogs(filePath string, lines int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()

	// Read all lines and get last N
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	return strings.Join(allLines[start:], "\n"), nil
}

func formatForOpenCode(source string, logs string) string {
	var sb strings.Builder
	sb.WriteString("# Error Context from dev-cli\n\n")
	sb.WriteString(fmt.Sprintf("**Source:** %s\n", source))
	sb.WriteString(fmt.Sprintf("**Exported:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("## Logs\n\n")
	sb.WriteString("```\n")
	sb.WriteString(logs)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Analyze these logs, identify any errors or issues, and suggest fixes.\n")
	return sb.String()
}

func saveForOpenCode(content string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	devCliDir := filepath.Join(homeDir, ".devlogs")
	if err := os.MkdirAll(devCliDir, 0755); err != nil {
		return "", err
	}

	savePath := filepath.Join(devCliDir, "last-error.md")
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return savePath, nil
}

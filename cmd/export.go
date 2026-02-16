package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"dev-cli/internal/storage"

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
		printError(fmt.Sprintf("%v", err))
		os.Exit(1)
	}

	if exportSave {
		savePath, err := storage.SaveErrorContext(source, logs)
		if err != nil {
			printError(fmt.Sprintf("Error saving: %v", err))
			os.Exit(1)
		}
		printSuccess(fmt.Sprintf("Saved to: %s", savePath))
		fmt.Printf("%sRun 'opencode' and use: @%s%s\n", colorCyan, savePath, colorReset)
	} else {
		fmt.Print(logs)
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

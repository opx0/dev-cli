# dev-cli

AI-powered command-line error analysis tool. Captures failed commands and explains why they failed using a local LLM (Ollama).

## Features

- 🔍 **Automatic error analysis** - Explains why commands failed
- 📝 **Fix suggestions** - Suggests commands to fix the issue
- 📊 **Command history** - Logs all commands with exit codes and output
- 🔎 **Query failures** - Filter by keyword, time range, or count

## Prerequisites

- **Go 1.21+** for building
- **Ollama** running locally (Docker or native)
  ```bash
  docker run -d -p 11434:11434 ollama/ollama
  ollama pull qwen2.5-coder:3b-instruct
  ```

## Installation

```bash
# Build
go build -o dev-cli .

# Add to PATH (optional)
sudo mv dev-cli /usr/local/bin/

# Setup shell integration (add to .zshrc)
echo 'eval "$(dev-cli hook zsh)"' >> ~/.zshrc
source ~/.zshrc
```

## Usage

### Automatic Analysis (via shell hook)

After installation, failed commands are automatically analyzed:

```bash
$ npm install
npm error: ENOENT: no such file or directory, package.json

❌ npm install (exit 254)
💡 Missing package.json in current directory
📝 Fix: npm init -y
   [Run Fix?] (y/n):
```

### Explicit Capture with `dcap`

Use `dcap` to capture command output for richer analysis:

```bash
dcap "npm install"    # Captures stdout/stderr for better context
dcap "prisma migrate" # Works with any command
```

### Query Historical Failures

```bash
# Last failure (default)
dev-cli rca

# Last N failures
dev-cli rca --last 5

# Filter by keyword
dev-cli rca --filter "npm"
dev-cli rca --filter "prisma"

# Filter by time
dev-cli rca --since "1h"    # Last hour
dev-cli rca --since "30m"   # Last 30 minutes

# Combine filters
dev-cli rca --filter "npm" --last 10 --since "24h"
```

### Interactive Mode

When running with `--interactive`, you'll be prompted to run suggested fixes:

```bash
dev-cli rca --last 1 --interactive
```

## Commands

| Command             | Description                             |
| ------------------- | --------------------------------------- |
| `dev-cli hook zsh`  | Print shell integration script          |
| `dev-cli log-event` | Log a command (used internally by hook) |
| `dev-cli rca`       | Analyze failures from log               |
| `dev-cli help`      | Show help                               |

## Configuration

| Environment Variable   | Default                     | Description         |
| ---------------------- | --------------------------- | ------------------- |
| `DEV_CLI_LOG_DIR`      | `~/.devlogs`                | Log file directory  |
| `DEV_CLI_OLLAMA_URL`   | `http://localhost:11434`    | Ollama API endpoint |
| `DEV_CLI_OLLAMA_MODEL` | `qwen2.5-coder:3b-instruct` | LLM model to use    |

## Log Format

Commands are logged to `~/.devlogs/history.jsonl`:

```json
{
  "command": "npm install",
  "exit_code": 254,
  "output": "...",
  "cwd": "/home/user/project",
  "duration_ms": 1234,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o dev-cli .

# Test with dev log directory
export DEV_CLI_LOG_DIR="./e2e/temp"
```

## License

MIT

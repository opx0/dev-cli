# dev-cli

AI-powered command-line error analysis tool. Captures failed commands and explains why they failed using a local LLM (Ollama).

## Features

- **×** Automatic error analysis - Explains why commands failed
- **$** Fix suggestions - Suggests executable commands to fix issues
- **→** Command history - Logs all commands with exit codes and output
- **⚠** Query failures - Filter by keyword, time range, or count

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

× npm install (exit 254)
  → Missing package.json
  $ npm init -y
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

## Output Format

```
× npm install (exit 254)     # Red × = failed command
  → Missing package.json     # Gray → = explanation
  $ npm init -y              # Green $ = fix command
   [Run Fix?] (y/n): y
   Running: npm init -y
   ✓ Fix applied             # Green ✓ = success
```

## Commands

| Command             | Description                             |
| ------------------- | --------------------------------------- |
| `dev-cli hook zsh`  | Print shell integration script          |
| `dev-cli log-event` | Log a command (used internally by hook) |
| `dev-cli rca`       | Analyze failures from log               |
| `dev-cli help`      | Show help                               |

### RCA Flags

| Flag                 | Description               | Example          |
| -------------------- | ------------------------- | ---------------- |
| `--last N`           | Analyze last N failures   | `--last 5`       |
| `--filter "keyword"` | Filter by command keyword | `--filter "npm"` |
| `--since "duration"` | Filter by time window     | `--since "1h"`   |
| `--interactive`      | Enable fix prompts        | `--interactive`  |

## Configuration

| Environment Variable   | Default                     | Description         |
| ---------------------- | --------------------------- | ------------------- |
| `DEV_CLI_LOG_DIR`      | `~/.devlogs`                | Log file directory  |
| `DEV_CLI_OLLAMA_URL`   | `http://localhost:11434`    | Ollama API endpoint |
| `DEV_CLI_OLLAMA_MODEL` | `qwen2.5-coder:3b-instruct` | LLM model to use    |

## Use Cases

### 1. Debug npm/Node.js Issues

```bash
$ dcap "npm install"
× npm install (exit 254)
  → Missing package.json
  $ npm init -y
```

### 2. Fix Permission Errors

```bash
$ dcap "cat /etc/shadow"
× cat /etc/shadow (exit 1)
  → Permission denied
  $ sudo cat /etc/shadow
```

### 3. Diagnose Git Problems

```bash
$ dcap "git push"
× git push (exit 128)
  → No configured push destination
  $ git remote add origin <url>
```

### 4. Missing Commands

```bash
$ dcap "prisma migrate"
× prisma (exit 127)
  → prisma is not recognized
  $ npm install -g prisma
```

### 5. Review Recent Failures

```bash
# What broke in the last hour?
dev-cli rca --since "1h" --last 10

# All npm issues today
dev-cli rca --filter "npm" --since "24h"
```

### 6. CI/CD Integration

```bash
# In your CI script
npm test || dev-cli log-event \
    --command "npm test" \
    --exit-code $? \
    --output "$(cat test.log)"
```

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

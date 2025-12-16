# dev-cli

**The Autonomous Agentic Terminal.**

`dev-cli` is an AI-powered companion for your terminal. It watches your commands, detects failures, explains them, and can even autonomously fix them.

## Features

- **🤖 Autonomous Agent** (`fix`): Auto-detects issues and executes fixes (with your permission).
- **🧠 Intelligent Research** (`ask`): Asks LLMs (Ollama/Perplexity) how to do things.
- **👀 Active Monitoring** (`watch`): Streams logs and alerts you on errors.
- **🔍 Root Cause Analysis** (`explain`): Deep dives into previous failures.
- **⚙️ Configurable**: Works with local (Ollama) or cloud (Perplexity) models.

## Installation

### Prerequisites

- **Go 1.21+**
- **Ollama** (for local AI):
  ```bash
  ollama run qwen2.5-coder:3b-instruct
  ```

### Build & Install

```bash
go build -o dev-cli .
sudo mv dev-cli /usr/local/bin/
```

### Shell Integration (Zsh)

Add this to your `~/.zshrc` to enable auto-capture:

```bash
eval "$(dev-cli init zsh)"
```

## Command Reference

### `fix`

**Usage**: `dev-cli fix [task]`
Autonomously attempts to solve a problem or execute a task.

- `[task]`: The natural language description of what you want to do.

### `ask`

**Usage**: `dev-cli ask [query...]`
Ask the AI for help with commands or general questions.

- `-n, --n <int>`: Number of commands to suggest (default 10).
- `--local`: Force use of local Ollama model even if cloud key is set.

### `explain` (alias: `why`, `rca`)

**Usage**: `dev-cli explain [flags]`
Analyze the last failed command or search history for failures.

- `-l, --last <int>`: Analyze the last N failures (default 1).
- `-f, --filter <string>`: Filter failures by command keyword.
- `-s, --since <duration>`: Filter by time (e.g., `1h`, `15m`).
- `-i, --interactive`: Enable interactive mode to run suggested fixes.

### `watch`

**Usage**: `dev-cli watch [flags]`
Monitor logs in real-time for errors.

- `--file <path>`: Path to a log file to watch.
- `--docker <container>`: Name or ID of a Docker container to watch.
- `--ai <backend>`: AI backend to use: `local` (default) or `cloud`.

### `ui`

**Usage**: `dev-cli ui`
Launch the interactive TUI (Mission Control) to view dashboard, monitor, and chat.

### `init` (alias: `hook`)

**Usage**: `dev-cli init [shell]`
Print the shell integration script.

- `[shell]`: Currently supports `zsh`.

### `log-event` (Internal)

**Usage**: `dev-cli log-event [flags]`
Used by the shell hook to log command execution.

- `--command <string>`: The command executed.
- `--exit-code <int>`: The exit code (0 = success).
- `--cwd <path>`: Working directory.
- `--duration-ms <int>`: Execution time in milliseconds.
- `--output <string>`: Captured command output.

## Database

`dev-cli` uses a local SQLite database to store command history.

- **Location**: `~/.devlogs/history.db` (override with `DEV_CLI_LOG_DIR`).
- **Schema**: Single `history` table.
- **Columns**: `id`, `timestamp`, `command`, `exit_code`, `output`, `cwd`, `duration_ms`, `session_id`, `details`.

The database is standard SQLite and can be queried with any SQLite client:

```bash
sqlite3 ~/.devlogs/history.db "SELECT command, exit_code FROM history WHERE exit_code != 0"
```

## Configuration

The CLI is configured via `~/.dev-cli/config.yaml` or Environment Variables.

| Variable                   | Description        | Default                     |
| -------------------------- | ------------------ | --------------------------- |
| `DEV_CLI_OLLAMA_URL`       | Ollama URL         | `http://localhost:11434`    |
| `DEV_CLI_OLLAMA_MODEL`     | Local Model        | `qwen2.5-coder:3b-instruct` |
| `DEV_CLI_PERPLEXITY_KEY`   | Perplexity API Key | `""`                        |
| `DEV_CLI_PERPLEXITY_MODEL` | Cloud Model        | `sonar-pro`                 |
| `DEV_CLI_LOG_DIR`          | Database Path      | `~/.devlogs`                |

## License

MIT

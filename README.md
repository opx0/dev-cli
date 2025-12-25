# dev-cli

**The Autonomous Agentic Terminal.**

`dev-cli` is an AI-powered companion for your terminal. It watches your commands, detects failures, explains them, and can even autonomously fix them.

## Features

- **🤖 Autonomous Agent** (`fix`): Auto-detects issues and executes fixes (with your permission).
- **🧠 Intelligent Research** (`ask`): Asks LLMs (Ollama/Perplexity) how to do things.
- **👀 Active Monitoring** (`watch`): Streams logs and alerts you on errors.
- **🔍 Root Cause Analysis** (`explain`): Deep dives into previous failures.
- **🖥️ Interactive TUI** (`ui`): Full-featured dashboard with agent, containers, and history tabs.
- **📊 Analytics** (`analytics`): Proactive debugging insights from command history.
- **🔄 Workflows** (`workflow`): Multi-step automation with rollback and checkpointing.
- **🏥 Doctor** (`doctor`): System health checks with auto-fix capabilities.
- **📤 Export** (`export`): Export logs formatted for OpenCode ingestion.
- **🔌 MCP Server** (`mcp-serve`): Model Context Protocol server for OpenCode integration.
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
Launch the interactive TUI (Mission Control).

**Tabs:**

- **Agent**: Chat interface with AI, execute commands.
- **Containers**: Docker container monitoring with live logs.
- **History**: Command history browser with details.

**Keybindings:**

| Key      | Action                     |
| -------- | -------------------------- |
| `1/2/3`  | Switch tabs                |
| `Tab`    | Cycle focus                |
| `i`      | Insert mode                |
| `Esc`    | Normal mode                |
| `Ctrl+o` | **Switch to OpenCode TUI** |
| `q`      | Quit                       |

### `doctor`

**Usage**: `dev-cli doctor [flags]`
Run health checks on all dev-cli dependencies and optionally fix issues.

**Checks:**

- Docker daemon status
- Docker Compose availability
- Ollama availability & model
- GPU/CUDA support
- Required directories (`~/.devlogs`)
- Network connectivity

**Flags:**

- `--fix`: Auto-fix issues where possible.
- `--quiet`: Suppress non-essential output.
- `--json`: Output results as JSON (for agent consumption).

### `workflow`

**Usage**: `dev-cli workflow <subcommand>`
Execute multi-step workflows defined in YAML files.

**Subcommands:**

- `run <file.yaml>`: Execute a workflow.
- `resume <run-id>`: Resume a paused or failed workflow.
- `list`: List recent workflow runs.
- `status <run-id>`: Show detailed status of a run.
- `rollback <run-id>`: Manually trigger rollback.

**Features:**

- Sequential step execution
- Conditional branching
- Automatic retry on failure
- Rollback capabilities
- Checkpoint/resume for long operations

### `analytics` (alias: `stats`, `insights`)

**Usage**: `dev-cli analytics [flags]`
Analyze command history to identify failure patterns and get proactive suggestions.

- `-c, --command <string>`: Analyze a specific command pattern.
- `-l, --limit <int>`: Number of patterns to show (default 10).
- `--json`: Output as JSON.

### `export`

**Usage**: `dev-cli export [flags]`
Export container or file logs in a format suitable for OpenCode.

- `--docker <container>`: Docker container to export logs from.
- `--file <path>`: Log file path to export from.
- `--lines <int>`: Number of log lines to export (default 50).
- `--save`: Save to `~/.devlogs/last-error.md` for OpenCode handoff.

### `mcp-serve`

**Usage**: `dev-cli mcp-serve`
Start a Model Context Protocol (MCP) server for OpenCode integration.

**OpenCode Configuration:**
Add this to your `opencode.json`:

```json
{
  "mcp": {
    "dev-cli": {
      "type": "local",
      "command": ["dev-cli", "mcp-serve"],
      "enabled": true
    }
  }
}
```

**Exposed Tools:**

- Query command history
- Find similar failures
- Get and store solutions
- Project fingerprinting

### `init` (alias: `hook`)

**Usage**: `dev-cli init [shell]`
Print the shell integration script.

- `[shell]`: Currently supports `zsh`.

## Database

`dev-cli` uses a local SQLite database to store command history.

- **Location**: `~/.devlogs/history.db` (override with `DEV_CLI_LOG_DIR`).
- **Schema**: Includes `history` and `workflow_runs` tables.

Query with any SQLite client:

```bash
sqlite3 ~/.devlogs/history.db "SELECT command, exit_code FROM history WHERE exit_code != 0"
```

## Configuration

The CLI is configured via `~/.devlogs/config.yaml` or Environment Variables.

| Variable                   | Description        | Default                     |
| -------------------------- | ------------------ | --------------------------- |
| `DEV_CLI_OLLAMA_URL`       | Ollama URL         | `http://localhost:11434`    |
| `DEV_CLI_OLLAMA_MODEL`     | Local Model        | `qwen2.5-coder:3b-instruct` |
| `DEV_CLI_PERPLEXITY_KEY`   | Perplexity API Key | `""`                        |
| `DEV_CLI_PERPLEXITY_MODEL` | Cloud Model        | `sonar-pro`                 |
| `DEV_CLI_LOG_DIR`          | Database Path      | `~/.devlogs`                |

## OpenCode Integration

`dev-cli` integrates with [OpenCode](https://github.com/opencode-ai/opencode) in multiple ways:

1. **TUI Handoff**: Press `Ctrl+o` in the dev-cli TUI to switch directly to OpenCode.
2. **MCP Server**: Use `dev-cli mcp-serve` to expose tools to OpenCode.
3. **Export**: Use `dev-cli export --save` to save logs for OpenCode ingestion.

## License

MIT

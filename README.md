# dev-cli

**AI-Powered DevOps Terminal Companion**

`dev-cli` is an autonomous AI agent for your terminal. It watches commands, explains failures, researches solutions, and can fix problems automatically using structured tool calling.

```
                    ┌─────────────────────────────────────────┐
  Human Terminal    │              dev-cli                    │
  ─────────────────►│  ask · explain · fix · watch · ui       │
                    └──────────────────┬──────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────────┐
                    │           internal/tools/               │
                    │  10 shared tools (file, git, docker...) │
                    └──────────────────┬──────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────────┐
  AI Agents         │              dev-mcp                    │
  (Cursor, Claude)  │     MCP Server for IDE integration      │
  ─────────────────►└─────────────────────────────────────────┘
```

## Quick Start

```bash
# Install
go build -o dev-cli . && sudo mv dev-cli /usr/local/bin/

# Start Ollama (required for local AI)
ollama run qwen2.5-coder:7b

# Shell integration (optional, enables auto-capture)
echo 'eval "$(dev-cli init zsh)"' >> ~/.zshrc

# Try it
dev-cli ask kubectl                    # Get useful commands
dev-cli explain                        # Why did my last command fail?
dev-cli fix "nginx container crashes"  # Let AI fix it
```

---

## Commands

### `fix` — Autonomous Agent

Launch an AI agent that analyzes issues and executes fixes using structured tool calling.

```bash
dev-cli fix "my nginx container keeps crashing"
dev-cli fix "disk is full on /var"
dev-cli fix --verbose "tests failing in auth module"
dev-cli fix --max-iterations 20 "complex refactoring"
```

**How it works:**
1. Agent analyzes the problem using diagnostic tools
2. Gathers information (reads files, searches code, inspects git)
3. Proposes fixes via `write_file` or `run_command`
4. Waits for your approval before destructive actions
5. Repeats until resolved or max iterations reached

**Flags:**
| Flag | Description |
|------|-------------|
| `-v, --verbose` | Show detailed progress |
| `--safe` | Require approval for destructive tools (default: true) |
| `--max-iterations N` | Limit tool-calling rounds (default: 10) |
| `--auto-approve` | Skip approval prompts (dangerous) |

### `ask` — AI Research

Get command suggestions or research DevOps topics.

```bash
dev-cli ask kubectl                  # Useful kubectl commands
dev-cli ask "how to resize LVM"      # Research a topic
dev-cli ask -n 5 docker              # Get 5 suggestions
dev-cli ask --local "nginx config"   # Force local model
```

### `explain` — Root Cause Analysis

Analyze why commands failed. Aliases: `why`, `rca`

```bash
dev-cli explain                      # Analyze last failure
dev-cli explain --last 3             # Analyze last 3 failures
dev-cli explain --filter npm         # Filter by keyword
dev-cli explain --since 1h           # Failures in last hour
dev-cli explain -i                   # Interactive (run suggested fix)
```

### `watch` — Log Monitor

Monitor logs in real-time with AI-powered error detection.

```bash
dev-cli watch --file /var/log/app.log
dev-cli watch --docker mycontainer
dev-cli watch --docker myapp --ai cloud   # Use Perplexity
```

### `ui` — Interactive Dashboard

Launch the TUI for monitoring, history, and chat.

```bash
dev-cli ui
```

### `workflow` — Multi-Step Automation

Execute predefined workflows with checkpointing and rollback.

```bash
dev-cli workflow run deploy.yaml
dev-cli workflow list
dev-cli workflow resume <run-id>
```

### `doctor` — System Health Check

Verify system dependencies and configuration.

```bash
dev-cli doctor
```

---

## Available Tools

The agent (`fix`) and MCP server (`dev-mcp`) share the same 10 tools:

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with optional line range |
| `read_dir` | List directory contents |
| `write_file` | Write or append to files |
| `run_command` | Execute shell commands with timeout |
| `search_codebase` | Search files by pattern (grep-like) |
| `query_docker` | Inspect containers, images, logs |
| `check_ports` | Check listening ports and processes |
| `git_info` | Get repository status and recent commits |
| `git_inspector` | Deep git analysis (blame, diff, log) |
| `package_info` | Query package managers (npm, pip, go) |

---

## MCP Server (dev-mcp)

`dev-mcp` exposes the same tools to AI-powered IDEs via the Model Context Protocol.

### Build

```bash
go build -o dev-mcp ./cmd/mcp
sudo mv dev-mcp /usr/local/bin/
```

### Configure Claude Desktop

Add to `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "dev-mcp": {
      "command": "/usr/local/bin/dev-mcp"
    }
  }
}
```

### Configure Cursor

Add to Cursor settings:

```json
{
  "mcp.servers": {
    "dev-mcp": {
      "command": "/usr/local/bin/dev-mcp"
    }
  }
}
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         cmd/                                    │
│  fix.go  ask.go  explain.go  watch.go  ui.go  workflow.go      │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────┴────────────────────────────────────┐
│                      internal/llm/                              │
│  Provider interface ◄── OllamaProvider, PerplexityProvider      │
│  Agent (tool-calling loop)                                      │
│  HybridClient (routes local/cloud)                              │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────┴────────────────────────────────────┐
│                     internal/tools/                             │
│  Registry (10 tools) ◄── Used by Agent + MCP                   │
│  Tool interface: Execute(ctx, params) → ToolResult              │
└────────────────────────────┬────────────────────────────────────┘
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ internal/       │ │ internal/       │ │ internal/       │
│ workflow/       │ │ executor/       │ │ mcp/            │
│ Engine,         │ │ Execute(),      │ │ MCP Server      │
│ Checkpointing,  │ │ Safety checks   │ │ (stdio)         │
│ Safe mode       │ │                 │ │                 │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

### Key Design Decisions

1. **Unified Tool Registry**: Both CLI (`dev-cli fix`) and MCP (`dev-mcp`) use the exact same tools from `internal/tools/`

2. **Provider Interface**: All LLM backends implement `Provider.ChatCompletion()` using the OpenAI-compatible API via `openai-go` SDK

3. **Structured Tool Calling**: Agent uses LLM tool calls (not bash string parsing) for reliable execution

4. **Safe Mode**: Destructive operations require user approval; patterns defined in `executor/safety.go`

5. **Workflow Integration**: Tool executions can route through `workflow.Engine` for checkpointing and rollback

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEV_CLI_OLLAMA_URL` | Ollama API endpoint | `http://localhost:11434` |
| `DEV_CLI_OLLAMA_MODEL` | Local model name | `qwen2.5-coder:7b` |
| `DEV_CLI_PERPLEXITY_KEY` | Perplexity API key | (none) |
| `DEV_CLI_PERPLEXITY_MODEL` | Cloud model name | `sonar-pro` |
| `DEV_CLI_LOG_DIR` | Database directory | `~/.devlogs` |

### Config File

`~/.devlogs/config.yaml`:

```yaml
ollama:
  url: http://localhost:11434
  model: qwen2.5-coder:7b

perplexity:
  api_key: pplx-xxxxx
  model: sonar-pro
```

---

## Database

Command history is stored in SQLite at `~/.devlogs/history.db`.

```bash
# Query failures
sqlite3 ~/.devlogs/history.db \
  "SELECT timestamp, command, exit_code FROM history WHERE exit_code != 0 LIMIT 10"
```

**Schema:**
- `id` — Auto-increment primary key
- `timestamp` — ISO 8601 timestamp
- `command` — Command string
- `exit_code` — Exit code (0 = success)
- `output` — Captured stdout/stderr
- `cwd` — Working directory
- `duration_ms` — Execution time
- `session_id` — Terminal session ID
- `details` — JSON metadata

---

## Development

```bash
# Build
go build -o dev-cli .
go build -o dev-mcp ./cmd/mcp

# Test
go test ./...

# Lint
golangci-lint run

# All checks
make check
```

### Project Structure

```
.
├── cmd/                    # CLI commands (cobra)
│   ├── mcp/               # MCP server entrypoint
│   └── *.go               # ask, explain, fix, watch, ui, workflow
├── internal/
│   ├── config/            # Configuration loading
│   ├── executor/          # Shell execution + safety
│   ├── llm/               # LLM providers + Agent
│   ├── mcp/               # MCP server implementation
│   ├── pipeline/          # Event bus
│   ├── storage/           # SQLite persistence
│   ├── tools/             # Shared tool registry (10 tools)
│   ├── tui/               # Terminal UI (bubbletea)
│   └── workflow/          # Workflow engine + checkpointing
├── main.go                # CLI entrypoint
└── go.mod
```

---

## License

MIT

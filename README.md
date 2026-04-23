# dev-cli

**AI-Powered DevOps Terminal Companion**

`dev-cli` is an autonomous AI agent for your terminal. It explains failures, researches solutions, and can fix problems automatically using structured tool calling with **safe mode** protection.

```
                    ┌─────────────────────────────────────────┐
  Human Terminal    │              dev-cli                    │
  ─────────────────►│       ask · explain · fix · ui          │
                    └──────────────────┬──────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────────┐
                    │           internal/tools/               │
                    │  10 shared tools (file, git, docker...) │
                    └─────────────────────────────────────────┘
```

## Quick Start

```bash
# Install
go build -o dev-cli . && sudo mv dev-cli /usr/local/bin/

# Start Ollama (required for local AI)
ollama run smallthinker

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

### `ui` — Interactive Dashboard

Launch the TUI for monitoring, history, and chat.

```bash
dev-cli ui
```

### `doctor` — System Health Check

Verify system dependencies and configuration.

```bash
dev-cli doctor
```

---

## Available Tools

The `fix` agent uses 10 structured tools for safe, reliable execution:

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

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         cmd/                                    │
│         fix.go  ask.go  explain.go  ui.go  doctor.go            │
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
│  Registry (10 tools) ◄── Used by Agent                          │
│  Tool interface: Execute(ctx, params) → ToolResult              │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              ▼                             ▼
┌─────────────────────────┐   ┌─────────────────────────┐
│ internal/workflow/      │   │ internal/executor/      │
│ Safe mode approval      │   │ Execute(),              │
│ (--safe flag support)   │   │ Safety checks           │
└─────────────────────────┘   └─────────────────────────┘
```

### Key Design Decisions

1. **Unified Tool Registry**: CLI commands use shared tools from `internal/tools/`

2. **Provider Interface**: All LLM backends implement `Provider.ChatCompletion()` using the OpenAI-compatible API via `openai-go` SDK

3. **Structured Tool Calling**: Agent uses LLM tool calls (not bash string parsing) for reliable execution

4. **Safe Mode**: Destructive operations require user approval; patterns defined in `executor/safety.go`

5. **Hybrid LLM**: Routes between local (Ollama) and cloud (Perplexity) based on task complexity

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEV_CLI_OLLAMA_URL` | Ollama API endpoint | `http://localhost:11434` |
| `DEV_CLI_OLLAMA_MODEL` | Local model name | `smallthinker` |
| `DEV_CLI_OPENAI_KEY` / `OPENAI_API_KEY` | OpenAI-compatible API key | (none) |
| `DEV_CLI_OPENAI_URL` / `OPENAI_BASE_URL` | OpenAI-compatible base URL | `https://api.openai.com/v1/` |
| `DEV_CLI_OPENAI_MODEL` / `OPENAI_MODEL` | Cloud model name | `gpt-4o-mini` |
| `DEV_CLI_PERPLEXITY_KEY` / `PERPLEXITY_API_KEY` | Perplexity API key | (none) |
| `DEV_CLI_PERPLEXITY_MODEL` | Perplexity model | `sonar-pro` |
| `DEV_CLI_FORCE_LOCAL` | Force local (skip cloud) when set | (unset) |
| `DEV_CLI_LOG_DIR` | Database directory | `~/.devlogs` (or `$XDG_DATA_HOME/dev-cli`) |

### Provider routing

`dev-cli fix` uses structured tool calling. Routing precedence:

1. `OPENAI_API_KEY` set (any OpenAI-compatible endpoint — OpenAI, OpenRouter, Groq, Together, vLLM, …) **and** `DEV_CLI_FORCE_LOCAL` unset → cloud.
2. Model name starts with `sonar*` and `PERPLEXITY_API_KEY` is set → Perplexity.
3. Otherwise → Ollama (local, no key required).

Small local models frequently mis-format function calls, so cloud is preferred when a tool-calling agent runs. Set `DEV_CLI_FORCE_LOCAL=1` to pin everything to Ollama.

### Config File

`~/.devlogs/config.yaml`:

```yaml
ollama:
  url: http://localhost:11434
  model: smallthinker

openai:
  api_key: sk-xxxxx
  base_url: https://api.openai.com/v1/
  model: gpt-4o-mini

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
│   └── *.go               # ask, explain, fix, ui, doctor
├── internal/
│   ├── config/            # Configuration loading
│   ├── executor/          # Shell execution + safety
│   ├── llm/               # LLM providers + Agent
│   ├── pipeline/          # Event bus
│   ├── storage/           # SQLite persistence
│   ├── tools/             # Shared tool registry (10 tools)
│   ├── tui/               # Terminal UI (bubbletea)
│   └── workflow/          # Safe mode support
├── main.go                # CLI entrypoint
└── go.mod
```

---

## License

MIT

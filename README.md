# dev-cli

`dev-cli` is a safety-first terminal companion for investigating command failures, explaining likely causes, and applying approved repairs. It combines deterministic local checks with optional local or cloud LLMs.

## What it does

- `explain` analyzes captured command failures, checking a local pattern database before using an LLM.
- `fix` runs a tool-using diagnostic agent. File writes and command execution require approval by default.
- `ask` provides command cheat sheets and DevOps research.
- `doctor` performs read-only dependency and provider checks.
- `ui` provides a focused Bubble Tea view of containers, logs, and command history.

The project deliberately does not provide workflow automation, code generation commands, PR/review helpers, runbook management, an MCP server, or a general infrastructure control plane.

## Install

Requirements: Go 1.25.4 or newer. Docker is optional; Ollama is optional when a cloud provider is configured.

```bash
git clone https://github.com/opx0/dev-cli.git
cd dev-cli
make build
./dev-cli --help
```

To install into the Go binary directory:

```bash
make install
```

## Quick start

Start with the read-only health check:

```bash
dev-cli doctor
```

Use a local Ollama server:

```bash
export DEV_CLI_OLLAMA_URL=http://localhost:11434
export DEV_CLI_OLLAMA_MODEL=smallthinker
dev-cli models pull smallthinker
```

Or configure an OpenAI-compatible provider:

```bash
export OPENAI_API_KEY=...
export OPENAI_MODEL=gpt-4o-mini
```

Then use the core workflows:

```bash
dev-cli ask "why is my container restarting?"
dev-cli explain --command "docker compose up" --exit-code 1 --output "..."
dev-cli fix --dry-run "find why the tests fail"
dev-cli fix --scope . "repair the failing tests"
dev-cli ui
```

## Command history integration

The optional shell hook records command metadata in a local SQLite database so `explain` can inspect recent failures. Add one of these to the relevant shell startup file:

```bash
# zsh
eval "$(dev-cli init zsh)"

# bash
eval "$(dev-cli init bash)"
```

The hook also defines `dcap`, which captures up to 10 KiB of command output for a richer explanation:

```bash
dcap docker compose up
dev-cli explain
```

Command and output data are sanitized for common credential patterns before storage. Review the shell hook with `dev-cli init zsh` or `dev-cli init bash` before enabling it.

## Safety model

Safety is the product boundary, not an optional mode.

- Diagnostic tools can run without confirmation.
- Mutating or unclassified agent tools fail closed and require explicit approval.
- `fix --dry-run` permits evidence gathering but blocks every mutation.
- `fix --scope <dir>` confines file writes to the resolved directory and blocks shell commands because their effects cannot be reliably contained.
- Dangerous command patterns and sensitive paths are blocked by the executor.
- Symlinks are resolved before path and scope checks.
- Cloud requests are sanitized for known credential patterns before transmission.
- `doctor` never applies fixes; it only reports recommended commands.

`--auto-approve` is intentionally dangerous. It removes interactive approval but does not bypass executor safety checks or `--scope`.

## Configuration

Configuration precedence is:

```text
environment variables > YAML file > defaults
```

Create and inspect the YAML file:

```bash
dev-cli config init
dev-cli config show
dev-cli config set ollama.model qwen2.5-coder:7b
```

API keys are refused by `config set` because command-line arguments can leak through shell history and process listings. Put credentials in environment variables or edit the `0600` config file directly.

| Setting | Environment variable | Default |
| --- | --- | --- |
| Ollama URL | `DEV_CLI_OLLAMA_URL` | `http://localhost:11434` |
| Ollama model | `DEV_CLI_OLLAMA_MODEL` | `smallthinker` |
| Force local LLM | `DEV_CLI_FORCE_LOCAL` | unset |
| OpenAI-compatible URL | `DEV_CLI_OPENAI_URL` or `OPENAI_BASE_URL` | `https://api.openai.com/v1/` |
| OpenAI key | `DEV_CLI_OPENAI_KEY` or `OPENAI_API_KEY` | unset |
| OpenAI model | `DEV_CLI_OPENAI_MODEL` or `OPENAI_MODEL` | `gpt-4o-mini` |
| Perplexity key | `DEV_CLI_PERPLEXITY_KEY` or `PERPLEXITY_API_KEY` | unset |
| Perplexity model | `DEV_CLI_PERPLEXITY_MODEL` | `sonar-pro` |
| Data directory | `DEV_CLI_LOG_DIR` | `~/.devlogs` or XDG data directory |
| Config file | `DEV_CLI_CONFIG` | `<data directory>/config.yaml` |

## Development

```bash
make install-lint
make check
make build
```

Useful narrower targets are `make test-short`, `make test`, and `make lint`.

The code is intentionally separated into a small command layer and focused internal packages:

```text
cmd/                 Cobra commands and presentation
internal/config/     runtime and YAML configuration
internal/errordb/    deterministic failure patterns
internal/executor/   process execution and safety policy
internal/health/     read-only doctor checks
internal/infra/      narrow Docker, Ollama, and port adapters
internal/llm/        provider routing, agent loop, sanitization
internal/storage/    local command history
internal/tools/      normalized agent tools
internal/tui/        focused Bubble Tea application
```

See [Plan.md](Plan.md) for the current product boundary and maintenance priorities.

## License

[MIT](LICENSE)

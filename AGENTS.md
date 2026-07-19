# AGENTS.md

## Product boundary

`dev-cli` is a safety-first DevOps terminal companion. Its core is failure investigation, deterministic evidence collection, optional AI explanation, approved remediation, and outcome review.

Core commands: `ask`, `explain`, `fix`, `doctor`, `ui`, `config`, `models`, and `version`.

Do not reintroduce workflow automation, runbooks, memory systems, PR/review/generation helpers, an MCP server, or a broad infrastructure dashboard without explicit product validation.

## Safety rules

Before file access or command execution, use the shared checks:

```go
fileCheck := executor.CheckFileSafety(path)
commandCheck := executor.CheckCommandSafety(command)
```

- Resolve symlinks before enforcing path scope.
- New or unclassified agent tools must fail closed and require approval.
- Dry-run may gather evidence but must never mutate state.
- Cloud requests must pass through request sanitization.
- Never expose Docker environment values or persist unsanitized command output.
- Extend the existing approval path; do not add a parallel safety system.

Key files:

- `internal/executor/safety.go`
- `internal/llm/agent.go`
- `internal/llm/sanitizer.go`
- `internal/tools/file_tools.go`
- `internal/tools/command_tools.go`

## Development

```bash
make install-lint
make check
make build
```

Keep the command tree and TUI small. Prefer deleting an obsolete abstraction over preserving speculative compatibility. Do not mutate unrelated user work.

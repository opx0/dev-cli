# AGENTS.md

> Quick context for AI agents. Full context at `.context/AGENT_CONTEXT.md`

## Project: dev-cli

AI-powered DevOps terminal companion with **safe mode** as the #1 priority.

## Current State

**Completed:**
- Feature validation research (HN, competitors)
- Killed: `watch`, `workflow` commands
- Safe Mode Hardening: 75+ dangerous patterns, 50+ sensitive file patterns
- Added `--dry-run` flag to `fix` command
- All tests passing

**Next:** Phase 1.3 - Local-First Excellence (Ollama improvements)

## Key Commands

```bash
dev-cli fix "issue"        # Autonomous agent
dev-cli fix --dry-run "x"  # Preview mode (safe)
dev-cli explain            # Root cause analysis
dev-cli ask "topic"        # AI research
dev-cli ui                 # Interactive TUI
```

## Safety (CRITICAL)

**Before any file/command operation, check safety:**
```go
// For files
check := executor.CheckFileSafety(path)
if !check.IsSafe { /* block */ }

// For commands
check := executor.CheckCommandSafety(cmd)
if !check.IsSafe { /* block */ }
```

**Key files:**
- `internal/executor/safety.go` - All safety patterns
- `internal/tools/file_tools.go` - File operations with safety
- `internal/tools/command_tools.go` - Command execution with safety

## Feature Decisions

| Decision | Features |
|----------|----------|
| **KILLED** | `watch`, `workflow`, templates |
| **DEFERRED** | MCP server |
| **CORE** | safe mode, explain, fix, ask, hybrid LLM |

## Build & Test

```bash
go build ./...  # Must pass
go test ./...   # Must pass
```

## Full Context

Read `.context/AGENT_CONTEXT.md` for:
- Complete progress history
- All safety patterns
- Full roadmap
- Coding guidelines

# dev-cli Agent Context

> **Purpose:** This file provides complete context for AI agents working on this project.
> **Location:** `.context/AGENT_CONTEXT.md` - Agents should read this first.
> **Last Updated:** January 2025

---

## Project Overview

**dev-cli** is an AI-powered DevOps terminal companion that helps developers:
- Explain command failures (`explain`)
- Research DevOps topics (`ask`)
- Autonomously fix issues (`fix`)
- Interactive dashboard (`ui`)

### Key Differentiators
1. **Safe Mode (9/9 priority)** - Trust is #1 gating factor for AI agent adoption
2. **Local-First (8/9)** - Ollama integration for privacy and offline use
3. **Hybrid LLM** - Routes between local (Ollama) and cloud (Perplexity) based on task

---

## What Has Been Done

### Phase 0: Feature Validation Research (COMPLETED)

Conducted evidence-based feature validation using:
- Hacker News analysis (6 query categories)
- Competitor analysis (aider, Claude Code, Cursor, Plandex)
- GitHub trends and community feedback

**Key Findings:**
| Finding | Evidence | Action |
|---------|----------|--------|
| Safe mode is critical | "Agents can rm -rf /, read your .env" - HN | Prioritize (9/9) |
| Local-first is differentiator | Lumos got 146 HN points | Prioritize (8/9) |
| MCP protocol weak adoption | 5x lower engagement than AI agents | Defer |
| Workflow YAML crowded | Terraform/Ansible dominate | Kill |
| Log monitoring crowded | Datadog/Loki dominate | Kill |

### Phase 1: Feature Cleanup (COMPLETED)

**Files Deleted:**
- `cmd/watch.go` - Log monitoring command (killed)
- `cmd/workflow.go` - Workflow YAML commands (killed)
- `internal/infra/logsink.go` - Log sink infrastructure (killed)

**Files Modified:**
- `cmd/root.go` - Removed watch from help text
- `README.md` - Removed killed features, updated architecture
- `internal/infra/docker.go` - Added inline LogEntry/LogSink types

**Files Created:**
- `docs/PROGRESS.md` - Complete project progress record
- `docs/ROADMAP.md` - Implementation roadmap

### Phase 2: Safe Mode Hardening (COMPLETED)

**Expanded `internal/executor/safety.go`:**
- From 48 lines to 280+ lines
- 75+ dangerous command patterns
- 50+ sensitive file patterns
- Sensitive directory protection

**Dangerous Command Categories (75+ patterns):**
```
File system:    rm -rf, rm -fr, shred, chmod -R 777, dd if=, mkfs
System:         shutdown, reboot, halt, poweroff, init 0/6
Git:            push --force, push -f, reset --hard, clean -fdx, 
                stash drop/clear, branch -D, reflog expire
Docker:         system prune, volume prune, rm -f, rmi -f
Kubernetes:     delete namespace, delete --all, delete pods --all
Database:       drop database/table/schema, truncate table, delete from
Network:        iptables -F, ufw disable
Package mgrs:   pip uninstall -y, npm uninstall -g, apt-get remove --purge
```

**Sensitive File Patterns (50+ patterns):**
```
Environment:    .env, .env.local, .env.production, *.env
Private keys:   *.pem, *.key, *.p12, *.pfx, id_rsa, id_ed25519
Credentials:    credentials.json/yaml, secrets.json/yaml, .netrc, .npmrc
Cloud configs:  *.tfvars, terraform.tfstate, .aws/credentials
Auth:           .htpasswd, passwd, shadow, authorized_keys
Application:    master.key, jwt_secret, api_key*
History:        .bash_history, .zsh_history, .mysql_history
```

**Sensitive Directories:**
```
.git, .ssh, .gnupg, .aws, .azure, .config/gcloud, .kube, node_modules
```

**Tool Integration:**
- `read_file` - Blocks reads of sensitive files
- `write_file` - Blocks writes to sensitive files  
- `run_command` - Blocks dangerous commands

**New Features:**
- `--dry-run` flag for `fix` command (preview mode)
- Safety severity levels: critical, danger, warning
- Comprehensive error messages explaining why blocked

**Tests Added:**
- `internal/executor/safety_test.go` - 150 lines, 42 test cases

---

## Current Project State

### Directory Structure
```
dev-cli/
├── .context/
│   └── AGENT_CONTEXT.md     # THIS FILE - Agent context
├── cmd/
│   ├── ask.go               # AI research command
│   ├── doctor.go            # System health check
│   ├── explain.go           # Root cause analysis
│   ├── fix.go               # Autonomous agent (--dry-run added)
│   ├── init.go              # Shell integration
│   ├── root.go              # CLI entrypoint
│   └── ui.go                # TUI dashboard
├── docs/
│   ├── PROGRESS.md          # Detailed progress record
│   └── ROADMAP.md           # Implementation roadmap
├── internal/
│   ├── config/              # Configuration loading
│   ├── executor/
│   │   ├── executor.go      # Shell execution
│   │   ├── safety.go        # Safety patterns (ENHANCED)
│   │   └── safety_test.go   # Safety tests (NEW)
│   ├── health/              # System health checks
│   ├── hook/                # Shell hooks
│   ├── infra/               # Docker, GPU, Ollama
│   ├── llm/
│   │   ├── agent.go         # AI agent (DryRun added)
│   │   ├── hybrid.go        # Hybrid LLM client
│   │   ├── ollama.go        # Ollama provider
│   │   └── perplexity.go    # Perplexity provider
│   ├── pipeline/            # Event bus
│   ├── storage/             # SQLite persistence
│   ├── tools/
│   │   ├── command_tools.go # run_command (safety added)
│   │   ├── file_tools.go    # read/write_file (safety added)
│   │   └── ...              # 10 total tools
│   ├── tui/                 # Terminal UI
│   └── workflow/
│       └── safemode.go      # Safe mode approval logic
├── main.go
├── go.mod
├── Makefile
└── README.md
```

### Available Commands
| Command | Description | Status |
|---------|-------------|--------|
| `dev-cli ask` | AI research and suggestions | Active |
| `dev-cli explain` | Root cause analysis | Active |
| `dev-cli fix` | Autonomous agent | Active (--dry-run added) |
| `dev-cli ui` | Interactive TUI | Active |
| `dev-cli doctor` | System health check | Active |
| `dev-cli init` | Shell integration | Active |

### Available Tools (for `fix` agent)
| Tool | Description | Safety |
|------|-------------|--------|
| `read_file` | Read file contents | Blocks sensitive files |
| `read_dir` | List directory | - |
| `write_file` | Write to file | Blocks sensitive files |
| `run_command` | Execute shell command | Blocks dangerous commands |
| `search_codebase` | Search files by pattern | - |
| `query_docker` | Inspect containers | - |
| `check_ports` | Check listening ports | - |
| `git_info` | Repository status | - |
| `git_inspector` | Deep git analysis | - |
| `package_info` | Query package managers | - |

### Key Files to Know
| File | Purpose |
|------|---------|
| `internal/executor/safety.go` | All safety patterns - CRITICAL |
| `internal/llm/agent.go` | AI agent implementation |
| `internal/tools/*.go` | Tool implementations |
| `cmd/fix.go` | Fix command with --dry-run |
| `internal/workflow/safemode.go` | Safe mode approval logic |

---

## What's Next (Roadmap)

### Phase 1.3: Local-First Excellence (NEXT)
**Priority: HIGH** (Score: 8/9)

Tasks:
- [ ] Improve Ollama error messages and recovery
- [ ] Add model auto-download if not present
- [ ] Add `dev-cli models` command to list/pull models
- [ ] Test with multiple models (qwen2.5-coder, codellama, deepseek-coder)
- [ ] Add offline mode indicator in output
- [ ] Document recommended models for different tasks

### Phase 2: Differentiation (2-4 weeks)

**2.1 Explain Enhancement:**
- [ ] Add `--context` flag for surrounding commands
- [ ] Improve error pattern recognition
- [ ] Add common error database (npm, docker, k8s, git)
- [ ] Add `--json` output

**2.2 Fix Agent Improvements:**
- [ ] Implement diff preview (cumulative diff review)
- [ ] Add rollback capability
- [ ] Improve iteration feedback
- [ ] Add `--scope` flag

**2.3 Ask Enhancement:**
- [ ] Add caching for common queries
- [ ] Add `--web` flag for documentation search
- [ ] Add clipboard integration (`--copy`)

### Phase 3: Polish (4-6 weeks)
- [ ] Improve TUI responsiveness
- [ ] Add progress indicators
- [ ] Add shell completions
- [ ] Increase test coverage to 80%+

### Phase 4: Growth (6+ weeks)
- [ ] Submit to awesome-go
- [ ] Write blog post about safe mode
- [ ] Create "Show HN" post
- [ ] Re-evaluate MCP Server adoption

---

## Key Decisions Made

### Features KILLED (Removed from codebase)
| Feature | Reason | Evidence |
|---------|--------|----------|
| `watch` (log monitoring) | Crowded market | Datadog, Loki dominate |
| `workflow` (YAML commands) | Crowded market | Terraform/Ansible dominate; YAML fatigue |
| Templates | No demand | No evidence of need |

### Features DEFERRED (Not implemented)
| Feature | Reason | Revisit |
|---------|--------|---------|
| MCP Server | Weak protocol adoption | 3-6 months |

### Features PRIORITIZED (Core)
| Feature | Score | Status |
|---------|-------|--------|
| Safe Mode | 9/9 | IMPLEMENTED |
| Explain | 8/9 | Active |
| Fix Agent | 8/9 | Active + --dry-run |
| Hybrid LLM | 8/9 | Active |
| Ask | 7/9 | Active |
| Shell Integration | 7/9 | Active |

---

## Coding Guidelines for This Project

### Safety First
- Always use `executor.CheckFileSafety()` before file operations
- Always use `executor.CheckCommandSafety()` before command execution
- Never bypass safety checks without explicit user approval

### Testing
- Run `go build ./...` and `go test ./...` after changes
- Add tests for new safety patterns
- Test coverage target: 80%+

### Code Style
- Follow existing patterns in the codebase
- Use `internal/` for non-public packages
- Keep tools in `internal/tools/`
- Keep LLM logic in `internal/llm/`

### Commits
- Reference the phase/task being worked on
- Include test results in commit messages

---

## Build & Test Commands

```bash
# Build
go build ./...

# Test all
go test ./...

# Test specific package
go test ./internal/executor/... -v

# Run the CLI
./dev-cli --help
./dev-cli fix --dry-run "test issue"
./dev-cli explain
./dev-cli ask "kubectl commands"
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEV_CLI_OLLAMA_URL` | Ollama API endpoint | `http://localhost:11434` |
| `DEV_CLI_OLLAMA_MODEL` | Local model name | `smallthinker` |
| `DEV_CLI_PERPLEXITY_KEY` | Perplexity API key | (none) |
| `DEV_CLI_PERPLEXITY_MODEL` | Cloud model name | `sonar-pro` |
| `DEV_CLI_LOG_DIR` | Database directory | `~/.devlogs` |

---

## Competitor Reference

| Competitor | Stars | Learn From | Don't Copy |
|------------|-------|------------|------------|
| aider | 42.9K | Git-native workflow | IDE features |
| Claude Code | 110K | Simple CLI | Plugin complexity |
| Cursor | - | Autonomy slider concept | IDE focus |
| Plandex | 15.2K | Diff sandbox, 2M context | Multi-file complexity |

---

## Quick Reference

**Safe Mode Patterns Location:** `internal/executor/safety.go`

**To add a new dangerous pattern:**
```go
// In DangerousPatterns slice
"new dangerous pattern",
```

**To add a new sensitive file pattern:**
```go
// In SensitiveFilePatterns slice
"*.sensitive",
```

**To add a new tool:**
1. Create in `internal/tools/`
2. Implement `Tool` interface
3. Register in `registry.go`
4. Add safety checks if needed

---

## Contact

- See `docs/PROGRESS.md` for detailed history
- See `docs/ROADMAP.md` for full roadmap
- Open issues on GitHub for questions

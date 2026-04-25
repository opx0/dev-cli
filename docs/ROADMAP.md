# dev-cli Implementation Roadmap

> Evidence-based roadmap derived from competitive analysis and community validation (Hacker News, GitHub trends, competitor research).

**Last Updated:** January 2025  
**Methodology:** Feature scoring (0-9 scale) based on market evidence, competitor analysis, and community feedback

---

## Executive Summary

### Features KILLED (Removed from Codebase)
| Feature | Score | Reason |
|---------|-------|--------|
| Log Monitoring (`watch`) | 2/9 | Crowded market (Datadog, Loki, etc.), not differentiated |
| Workflow YAML Commands | 2/9 | Terraform/Ansible dominate; YAML fatigue is real |
| Templates System | 1/9 | No evidence of demand; cookiecutter exists |

### Features DEFERRED (Not Implemented Yet)
| Feature | Score | Reason |
|---------|-------|--------|
| MCP Server (`dev-mcp`) | 2/9 | Protocol adoption weak (5x lower engagement than AI agents); wait for ecosystem maturity |

### Features KEPT (Core Differentiators)
| Feature | Score | Reason |
|---------|-------|--------|
| Safe Mode | 9/9 | Trust is #1 gating factor for AI agent adoption |
| Explain/RCA | 8/9 | Context switching is #1 developer pain point |
| Fix (Autonomous Agent) | 8/9 | Plandex v2 had highest HN engagement (257 pts) |
| Hybrid LLM (Local+Cloud) | 8/9 | Ollama/local-first tools get 5x more engagement |
| Ask (Research) | 7/9 | Complements explain; reduces context switching |
| Shell Integration | 7/9 | Enables automatic failure capture |

---

## Phase 1: Foundation (Current → 2 weeks)

### 1.1 Stabilize Core Features
**Priority: CRITICAL** - COMPLETED

- [x] Remove killed features from codebase
  - [x] `cmd/watch.go` - log monitoring
  - [x] `cmd/workflow.go` - workflow YAML commands
  - [x] `internal/infra/logsink.go` - log sink infrastructure
- [x] Update `cmd/root.go` help text
- [x] Verify build passes
- [x] Verify tests pass
- [x] Update README.md to reflect current state
- [ ] Clean up orphaned code in `internal/workflow/` (keep safemode.go)

### 1.2 Safe Mode Hardening
**Priority: CRITICAL** (Score: 9/9) - COMPLETED

Evidence from Hacker News:
> "Agents can rm -rf /, read your .env, push to production" - Trust is gating factor for adoption

**Tasks:**
- [x] Audit `executor/safety.go` patterns (expanded to 75+ patterns)
- [x] Add protection for sensitive files (`.env`, `credentials.json`, `*.pem`, `*.key`) - 50+ patterns
- [x] Add protection for dangerous git operations (`push --force`, `reset --hard`)
- [x] Add safety checks to `read_file`, `write_file`, `run_command` tools
- [x] Add `--dry-run` flag to `fix` command
- [x] Add safety tests (`internal/executor/safety_test.go`)
- [ ] Implement "diff sandbox" concept from Plandex (show changes before applying) - DEFERRED to Phase 2
- [ ] Create comprehensive safe mode documentation

### 1.3 Local-First Excellence
**Priority: HIGH** (Score: 8/9) - COMPLETED

Evidence: Lumos (Ollama CLI) got 146 HN points - highest engagement for local AI tools

**Tasks:**
- [x] Improve Ollama error messages and recovery
- [x] Add model auto-download if not present
- [x] Add `dev-cli models` command to list/pull models
- [x] Test with multiple models (qwen2.5-coder, codellama, deepseek-coder) — documented in MODELS.md
- [x] Add offline mode indicator in output
- [x] Document recommended models for different tasks — `docs/MODELS.md`

---

## Phase 2: Differentiation (2-4 weeks)

### 2.1 Explain Enhancement
**Priority: HIGH** (Score: 8/9) - COMPLETED

Evidence: Context switching is #1 developer pain point

**Tasks:**
- [x] Add `--context` flag to include surrounding commands
- [x] Improve error pattern recognition — built-in 50+ pattern database
- [x] Add common error database (npm, docker, k8s, git, go, python)
- [ ] Integrate with shell history for better context
- [x] Add `--json` output for programmatic use

### 2.2 Fix Agent Improvements
**Priority: HIGH** (Score: 8/9) - COMPLETED

Evidence: Plandex v2 (257 pts, 81 comments) - "diff sandbox" is key differentiator

**Tasks:**
- [x] Implement cumulative diff review (show all changes before commit)
- [x] Add rollback capability for file changes
- [x] Improve iteration feedback (show progress, what was tried)
- [x] Add `--scope` flag to limit changes to specific directories
- [ ] Add git auto-commit after successful fix (optional)
- [ ] Improve tool selection heuristics

### 2.3 Ask Enhancement
**Priority: MEDIUM** (Score: 7/9)

**Tasks:**
- [ ] Add caching for common queries
- [ ] Improve command suggestions with examples
- [ ] Add `--web` flag to search documentation sites
- [ ] Add clipboard integration (`--copy`)

---

## Phase 3: Polish (4-6 weeks)

### 3.1 Developer Experience
**Priority: MEDIUM**

**Tasks:**
- [ ] Improve TUI (`ui` command) responsiveness
- [ ] Add progress indicators for long operations
- [ ] Add color themes
- [ ] Improve error messages across all commands
- [ ] Add shell completions (zsh, bash, fish)

### 3.2 Documentation
**Priority: MEDIUM**

**Tasks:**
- [ ] Create `docs/` site with examples
- [ ] Add video demos for each command
- [ ] Document all flags and configuration options
- [ ] Add troubleshooting guide
- [ ] Add contribution guide

### 3.3 Testing
**Priority: MEDIUM**

**Tasks:**
- [ ] Increase test coverage to 80%+
- [ ] Add integration tests for LLM interactions (mocked)
- [ ] Add E2E tests for CLI commands
- [ ] Set up CI/CD with GitHub Actions

---

## Phase 4: Growth (6+ weeks)

### 4.1 Community
**Priority: LOW**

**Tasks:**
- [ ] Submit to awesome-go list
- [ ] Write blog post about safe mode approach
- [ ] Create HN "Show HN" post
- [ ] Engage with AI CLI community (aider, plandex discussions)

### 4.2 MCP Server (Re-evaluate)
**Priority: LOW** (Score: 2/9 - may increase)

**Decision Point:** Re-evaluate MCP adoption in 3-6 months
- Monitor Claude Desktop and Cursor MCP adoption rates
- Check for production use cases in enterprise
- If adoption increases, implement `dev-mcp`

**Current State:**
- Build targets exist in Makefile and .goreleaser.yaml
- No actual implementation in `cmd/mcp/`
- Keep references but don't prioritize

---

## Competitor Insights

### What to Learn From Each

| Competitor | Key Insight | Apply To dev-cli |
|------------|-------------|------------------|
| **aider** (42.9K stars) | Git-native workflow, supports 100+ languages | Improve git integration in `fix` |
| **Claude Code** (110K stars) | Plugin architecture, natural language | Keep CLI simple, don't over-engineer |
| **Cursor** | "Autonomy slider" (Tab → Cmd+K → Agent) | Consider `--autonomy` levels for `fix` |
| **Plandex** (15.2K stars) | Diff sandbox, 2M token context | Implement diff preview, improve context |

### What NOT to Do

1. **Don't add IDE integration** - Cursor dominates this; stay terminal-native
2. **Don't add workflow YAML** - Terraform/Ansible dominate; YAML fatigue is real
3. **Don't add log monitoring** - Datadog/Loki dominate; not differentiated
4. **Don't rush MCP** - Protocol adoption is weak; wait for ecosystem

---

## Success Metrics

### Phase 1 (Foundation)
- [ ] Zero crashes reported
- [ ] Safe mode catches 100% of dangerous operations
- [ ] Works offline with Ollama

### Phase 2 (Differentiation)
- [ ] `explain` reduces context switching by showing relevant fixes
- [ ] `fix` success rate > 70% for common issues
- [ ] Local-first users can work without cloud API

### Phase 3 (Polish)
- [ ] < 2 second startup time
- [ ] Documentation covers all use cases
- [ ] Test coverage > 80%

### Phase 4 (Growth)
- [ ] 100+ GitHub stars
- [ ] 10+ community contributors
- [ ] Featured in at least one "awesome" list

---

## Appendix: Evidence Sources

### Hacker News Analysis (January 2025)
- "AI CLI tools" - 6 stories analyzed
- "aider AI" - Highest engagement for code editing (42.9K GitHub stars)
- "Ollama" - Local-first tools get 5x more engagement
- "MCP protocol" - Average 3.5 pts/story vs 17.8 for AI agents
- "AI debugging" - Explains #1 pain point: context switching
- "workflow automation" - Terraform/Ansible dominate

### Competitor GitHub Stars (January 2025)
- Claude Code: ~110K stars
- aider: 42.9K stars
- Plandex: 15.2K stars
- dev-cli: Building...

### Key Quotes
> "Agents can rm -rf /, read your .env, push to production" - HN comment on AI safety
> "The diff sandbox is what makes this usable" - HN comment on Plandex
> "I just want it to work offline" - HN comment on AI CLI tools

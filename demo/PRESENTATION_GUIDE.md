# dev-cli Presentation Guide

## Quick Start

```bash
cd demo
chmod +x *.sh
./demo_runner.sh
```

---

## Demo Order (Recommended)

| #   | Feature     | Time  | Key Talking Points                     |
| --- | ----------- | ----- | -------------------------------------- |
| 1   | Shell Hook  | 2 min | Automatic capture, zero-config         |
| 2   | Ask AI      | 2 min | Natural language queries, local/cloud  |
| 3   | Explain/RCA | 3 min | Root cause analysis, filters           |
| 4   | Watch Logs  | 3 min | Real-time monitoring, instant insights |
| 5   | Fix Agent   | 2 min | Autonomous fixes, human approval       |
| 6   | Doctor      | 2 min | Health checks, auto-fix                |
| 7   | Analytics   | 2 min | Proactive insights, patterns           |
| 8   | Workflow    | 3 min | Multi-step, rollback, checkpoints      |
| 9   | TUI         | 3 min | Full dashboard, containers             |

**Total: ~22 minutes**

---

## Key Features to Highlight

### 🔌 Shell Hook

- **Zero configuration** - just add to `.zshrc`
- **Captures everything** - command, exit code, duration, output
- **SQLite storage** - queryable history

### 🧠 AI Integration

- **Local AI (Ollama)** - privacy, no API keys
- **Cloud AI (Perplexity)** - when you need more power
- **Seamless switching** - just set env var

### 🔍 RCA (Root Cause Analysis)

- **Filters** - by command, time, limit
- **Aliases** - `explain`, `why`, `rca`
- **Interactive mode** - run fixes directly

### 🏥 Doctor

- **7 health checks** - Docker, Ollama, GPU, network
- **JSON output** - for agent consumption
- **Auto-fix** - one flag fixes everything

### 🔄 Workflows

- **YAML definitions** - human readable
- **Conditionals** - dynamic execution paths
- **Rollback** - automatic on failure
- **Checkpoints** - resume long operations

---

## Tips for Live Demo

1. **Pre-build the CLI**: Run `go build -o dev-cli .` before demo
2. **Start Ollama**: Make sure local AI is running
3. **Seed data**: Run a few log-event commands for analytics
4. **Use temp dir**: Set `DEV_CLI_LOG_DIR` to avoid affecting real data
5. **Have fallbacks**: Screenshots/recordings if something fails

---

## Troubleshooting

| Issue                        | Solution                                |
| ---------------------------- | --------------------------------------- |
| "Ollama not responding"      | `docker start ollama` or `ollama serve` |
| "No command history"         | Run demo 01 first to seed data          |
| "Watch not detecting errors" | Check log file path, wait 2-3 seconds   |
| "Workflow fails to parse"    | Validate YAML syntax                    |

---

## Files Created

```
demo/
├── demo_runner.sh      # Interactive menu
├── common.sh           # Shared helpers
├── 01_shell_hook.sh    # Shell integration
├── 02_ask_ai.sh        # AI queries
├── 03_explain_rca.sh   # Root cause analysis
├── 04_watch_logs.sh    # Log monitoring
├── 05_fix_agent.sh     # Autonomous fixes
├── 06_doctor.sh        # Health checks
├── 07_analytics.sh     # Proactive insights
├── 08_workflow.sh      # Workflow automation
├── 09_tui.sh           # Interactive TUI
├── PRESENTATION_GUIDE.md
└── workflows/
    ├── deploy_demo.yaml
    └── test_suite.yaml
```

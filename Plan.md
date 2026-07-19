# dev-cli plan

## Current product

The supported product is a safety-first investigation loop:

```text
capture failure → collect deterministic evidence → explain → approve a repair → verify
```

The maintained command surface is `ask`, `explain`, `fix`, `doctor`, `ui`, `config`, `models`, and `version`. The TUI is limited to container/log inspection and command history.

## Current priorities

1. Preserve fail-closed tool approval, path safety, credential sanitization, and dry-run guarantees.
2. Improve deterministic diagnosis and verification before expanding AI behavior.
3. Keep local Ollama and optional cloud providers behind the same provider boundary.
4. Add focused tests for safety-critical behavior and provider failure modes.
5. Reject features that duplicate mature external tools or require permanent main-TUI surface.

## Removed scope

The following are intentionally not part of the product: workflow YAML, runbook storage, long-term memory, PR/review/commit generation, generic code generation, test generation, export subsystems, an MCP binary, automatic infrastructure setup, diff rollback, pipeline visualization, and a multi-purpose infrastructure dashboard.

## Gate for new integrations

A new integration must provide evidence-based diagnosis or safe remediation that existing tools do not already offer. It must be optional, isolated from unrelated commands, useful without AI, testable without the external service, and absent from the main TUI until validated.

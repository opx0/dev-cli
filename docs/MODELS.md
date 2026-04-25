# Recommended Models for dev-cli

> Guide for choosing the right local model for different tasks.

## Quick Reference

| Task | Recommended Model | Size | Why |
|------|-------------------|------|-----|
| **Fix (Agent)** | `qwen2.5-coder:7b` | ~4.7 GB | Best tool-calling accuracy for its size |
| **Explain** | `smallthinker` | ~2 GB | Fast, good at error analysis |
| **Ask (Research)** | `qwen2.5-coder:7b` | ~4.7 GB | Good instruction following |
| **Ask (Quick)** | `smallthinker` | ~2 GB | Fastest response time |
| **General (Balanced)** | `deepseek-coder-v2:16b` | ~8.9 GB | Best quality if you have RAM |

## Model Details

### smallthinker (Default)
```bash
dev-cli models pull smallthinker
```
- **Size:** ~2 GB
- **Best for:** Quick explanations, cheat sheets, log analysis
- **Pros:** Fast inference, low memory, good enough for most tasks
- **Cons:** Struggles with complex multi-step tool calling
- **VRAM:** ~2-3 GB

### qwen2.5-coder:7b (Recommended for Agent)
```bash
dev-cli models pull qwen2.5-coder:7b
```
- **Size:** ~4.7 GB
- **Best for:** `fix` command (autonomous agent), code generation
- **Pros:** Excellent function-call formatting, code-aware
- **Cons:** Slower than smallthinker, needs more VRAM
- **VRAM:** ~5-6 GB

### codellama:7b
```bash
dev-cli models pull codellama:7b
```
- **Size:** ~3.8 GB
- **Best for:** Code explanation, debugging
- **Pros:** Strong code understanding, Meta's quality
- **Cons:** Weaker at tool-calling than qwen2.5-coder
- **VRAM:** ~5 GB

### deepseek-coder-v2:16b
```bash
dev-cli models pull deepseek-coder-v2:16b
```
- **Size:** ~8.9 GB
- **Best for:** Complex debugging, large codebase analysis
- **Pros:** Highest quality output, best code reasoning
- **Cons:** Needs 16 GB+ VRAM, slower inference
- **VRAM:** ~12-16 GB

---

## How to Switch Models

### Per-session (environment variable)
```bash
export DEV_CLI_OLLAMA_MODEL=qwen2.5-coder:7b
dev-cli fix "tests failing"
```

### Permanent (config file)
```bash
dev-cli config set ollama.model qwen2.5-coder:7b
```

### Per-command (one-shot)
```bash
DEV_CLI_OLLAMA_MODEL=codellama:7b dev-cli explain
```

---

## GPU Memory Guide

Your GPU: **NVIDIA GeForce RTX 3050 Laptop (4 GB VRAM)**

| Model | Fits in 4 GB? | Notes |
|-------|---------------|-------|
| smallthinker | ✅ Yes | Room to spare |
| qwen2.5-coder:7b | ⚠️ Tight | Works but may offload to CPU |
| codellama:7b | ⚠️ Tight | Similar to qwen2.5-coder |
| deepseek-coder-v2:16b | ❌ No | Needs 16 GB+ VRAM |

> **Tip:** With 4 GB VRAM, use `smallthinker` for speed or `qwen2.5-coder:7b` when
> you need better tool-calling accuracy. The 7b models will partially offload to CPU
> but still run faster than pure CPU inference.

---

## Cloud Fallback

When a task exceeds local model capabilities (e.g., complex multi-file refactoring),
dev-cli automatically routes to a cloud provider if configured:

```bash
# OpenAI-compatible endpoint (recommended for fix agent)
export DEV_CLI_OPENAI_KEY=sk-...
export DEV_CLI_OPENAI_MODEL=gpt-4o-mini

# Perplexity (recommended for ask/research)
export DEV_CLI_PERPLEXITY_KEY=pplx-...
```

The `fix` command prefers cloud for tool-calling when available.
Use `--local` flag or `DEV_CLI_FORCE_LOCAL=1` to stay offline.

---

## Managing Models

```bash
# List installed models
dev-cli models list

# Pull a new model
dev-cli models pull qwen2.5-coder:7b

# Remove a model (free disk space)
dev-cli models rm codellama:7b
```

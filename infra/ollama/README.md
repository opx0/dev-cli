# Ollama for dev-cli

Production-ready Ollama deployment for dev-cli's local AI features.

## Quick Start

```bash
cd infra/ollama

# Optional: choose which model is auto-pulled inside Docker
export OLLAMA_MODEL=smallthinker

# Start Ollama API + model init
docker compose up -d
```

This stack exposes Ollama API at `http://localhost:11434` and auto-pulls
`${OLLAMA_MODEL}` (defaults to `smallthinker`) using a one-shot init container.

If you already have models on the host and want Docker to use them directly,
mount your host Ollama directory before startup:

```bash
export OLLAMA_MODELS_PATH="$HOME/.ollama"
docker compose up -d
```

## Use `smallthinker` in dev-cli

Point dev-cli to the Docker API and set the model name:

```bash
export DEV_CLI_OLLAMA_URL=http://localhost:11434
export DEV_CLI_OLLAMA_MODEL=smallthinker
```

Then run:

```bash
dev-cli doctor
dev-cli ask --local "test local model"
```

## Requirements

- Docker with NVIDIA GPU support (nvidia-container-toolkit)
- NVIDIA GPU with CUDA support

## CPU-Only Mode

If you don't have an NVIDIA GPU, remove the `deploy` section from docker-compose.yml:

```yaml
# Remove this section:
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: 1
          capabilities: [gpu]
```

## Pull a Model

```bash
docker exec -it ollama ollama pull llama3.2
```

Or with API (no interactive shell):

```bash
curl -s http://localhost:11434/api/pull \
  -d '{"name":"smallthinker"}'
```

## Verify

```bash
curl http://localhost:11434/api/tags
```

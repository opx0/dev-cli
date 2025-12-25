# Ollama for dev-cli

Production-ready Ollama deployment for dev-cli's local AI features.

## Quick Start

```bash
cd infra/ollama
docker compose up -d
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

## Verify

```bash
curl http://localhost:11434/api/tags
```

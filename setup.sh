#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# dev-cli Setup Script — One command to get everything running
#
# Usage:
#   ./setup.sh           Full setup (Docker + Ollama + GPU + build + test)
#   ./setup.sh --check   Just check what's missing (no changes)
#   ./setup.sh --skip-gpu  Skip GPU/NVIDIA setup (CPU-only Ollama)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
DIM='\033[2m'
BOLD='\033[1m'
RESET='\033[0m'

# ── Helpers ───────────────────────────────────────────────────────────────────
step()  { echo -e "\n${CYAN}▸${RESET} ${BOLD}$1${RESET}"; }
ok()    { echo -e "  ${GREEN}✓${RESET} $1"; }
warn()  { echo -e "  ${YELLOW}⚠${RESET} $1"; }
fail()  { echo -e "  ${RED}✗${RESET} $1"; }
info()  { echo -e "  ${DIM}$1${RESET}"; }

CHECK_ONLY=false
SKIP_GPU=false
for arg in "$@"; do
  case "$arg" in
    --check)    CHECK_ONLY=true ;;
    --skip-gpu) SKIP_GPU=true ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/infra/ollama/docker-compose.yml"
OLLAMA_MODEL="${OLLAMA_MODEL:-smallthinker}"
ERRORS=0

echo -e "${BOLD}━━━ dev-cli Setup ━━━${RESET}"
echo -e "${DIM}$(date '+%Y-%m-%d %H:%M:%S')${RESET}"

# ─────────────────────────────────────────────────────────────────────────────
# Step 1: Docker Daemon
# ─────────────────────────────────────────────────────────────────────────────
step "1/8 Docker Daemon"

if command -v docker &>/dev/null; then
  ok "Docker binary found: $(docker --version | head -1)"
else
  fail "Docker not installed"
  info "Install: sudo pacman -S docker"
  ERRORS=$((ERRORS + 1))
fi

if docker ps &>/dev/null; then
  ok "Docker daemon is running"
else
  warn "Docker daemon is not running"
  if $CHECK_ONLY; then
    info "Run: sudo systemctl start docker && sudo systemctl enable docker"
    ERRORS=$((ERRORS + 1))
  else
    info "Starting Docker daemon..."
    sudo systemctl start docker
    sudo systemctl enable docker
    sleep 2
    if docker ps &>/dev/null; then
      ok "Docker daemon started and enabled"
    else
      fail "Could not start Docker daemon"
      ERRORS=$((ERRORS + 1))
    fi
  fi
fi

# Check docker group membership
if groups | grep -q docker; then
  ok "User is in docker group"
else
  warn "User not in docker group (commands may need sudo)"
  if ! $CHECK_ONLY; then
    info "Adding user to docker group..."
    sudo usermod -aG docker "$(whoami)"
    ok "Added to docker group (re-login or run 'newgrp docker' to apply)"
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 2: Docker Compose
# ─────────────────────────────────────────────────────────────────────────────
step "2/8 Docker Compose"

if docker compose version &>/dev/null; then
  ok "Docker Compose: $(docker compose version --short 2>/dev/null || echo 'installed')"
else
  warn "Docker Compose plugin not installed"
  if $CHECK_ONLY; then
    info "Install: sudo pacman -S docker-compose"
    ERRORS=$((ERRORS + 1))
  else
    info "Installing docker-compose..."
    sudo pacman -S --noconfirm docker-compose
    if docker compose version &>/dev/null; then
      ok "Docker Compose installed"
    else
      fail "Could not install docker-compose"
      ERRORS=$((ERRORS + 1))
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 3: NVIDIA GPU
# ─────────────────────────────────────────────────────────────────────────────
step "3/8 NVIDIA GPU"

if $SKIP_GPU; then
  info "Skipping GPU setup (--skip-gpu)"
else
  if nvidia-smi &>/dev/null; then
    GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
    ok "GPU detected: ${GPU_NAME}"
  else
    warn "No NVIDIA GPU detected or driver not loaded"
    info "Ollama will run in CPU-only mode"
    SKIP_GPU=true
  fi

  if ! $SKIP_GPU; then
    if command -v nvidia-ctk &>/dev/null; then
      ok "nvidia-container-toolkit installed"
    else
      warn "nvidia-container-toolkit not installed"
      if $CHECK_ONLY; then
        info "Install: sudo pacman -S nvidia-container-toolkit"
        ERRORS=$((ERRORS + 1))
      else
        info "Installing nvidia-container-toolkit..."
        sudo pacman -S --noconfirm nvidia-container-toolkit
        if command -v nvidia-ctk &>/dev/null; then
          ok "nvidia-container-toolkit installed"
        else
          fail "Could not install nvidia-container-toolkit"
          ERRORS=$((ERRORS + 1))
        fi
      fi
    fi

    # Configure Docker NVIDIA runtime
    if docker info 2>/dev/null | grep -q "nvidia"; then
      ok "NVIDIA runtime configured in Docker"
    else
      if ! $CHECK_ONLY && command -v nvidia-ctk &>/dev/null; then
        info "Configuring NVIDIA runtime for Docker..."
        sudo nvidia-ctk runtime configure --runtime=docker
        sudo systemctl restart docker
        sleep 2
        ok "NVIDIA runtime configured"
      fi
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 4: Go Toolchain
# ─────────────────────────────────────────────────────────────────────────────
step "4/8 Go Toolchain"

if command -v go &>/dev/null; then
  ok "Go: $(go version | awk '{print $3}')"
else
  fail "Go not installed"
  ERRORS=$((ERRORS + 1))
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 5: Build dev-cli
# ─────────────────────────────────────────────────────────────────────────────
step "5/8 Build"

if $CHECK_ONLY; then
  info "Skipping build (check-only mode)"
else
  info "Building dev-cli..."
  if (cd "$SCRIPT_DIR" && go build -o dev-cli . 2>&1); then
    ok "dev-cli binary built"
  else
    fail "Build failed"
    ERRORS=$((ERRORS + 1))
  fi

  info "Building dev-mcp..."
  if (cd "$SCRIPT_DIR" && go build -o dev-mcp ./cmd/mcp/ 2>&1); then
    ok "dev-mcp binary built"
  else
    fail "dev-mcp build failed"
    ERRORS=$((ERRORS + 1))
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 6: Ollama Container
# ─────────────────────────────────────────────────────────────────────────────
step "6/8 Ollama Container"

if ! docker ps &>/dev/null; then
  warn "Docker not running — skipping Ollama setup"
  ERRORS=$((ERRORS + 1))
else
  if $CHECK_ONLY; then
    if curl -sf http://localhost:11434/api/tags &>/dev/null; then
      ok "Ollama API is responding"
    else
      warn "Ollama not running"
      info "Start: docker compose -f infra/ollama/docker-compose.yml up -d"
    fi
  else
    # If GPU not available, strip the deploy section on the fly
    if $SKIP_GPU; then
      info "Starting Ollama (CPU-only)..."
      # Create a temporary override that removes GPU requirement
      cat > "${SCRIPT_DIR}/infra/ollama/docker-compose.override.yml" <<'EOF'
services:
  ollama:
    deploy: {}
EOF
      docker compose -f "$COMPOSE_FILE" -f "${SCRIPT_DIR}/infra/ollama/docker-compose.override.yml" up -d
    else
      info "Starting Ollama (GPU-accelerated)..."
      docker compose -f "$COMPOSE_FILE" up -d
    fi

    # Wait for Ollama to be healthy
    info "Waiting for Ollama to be ready..."
    ATTEMPTS=0
    MAX_ATTEMPTS=30
    while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
      if curl -sf http://localhost:11434/api/tags &>/dev/null; then
        ok "Ollama API is ready"
        break
      fi
      ATTEMPTS=$((ATTEMPTS + 1))
      sleep 2
    done

    if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
      fail "Ollama did not start in time"
      ERRORS=$((ERRORS + 1))
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 7: Model
# ─────────────────────────────────────────────────────────────────────────────
step "7/8 Model: ${OLLAMA_MODEL}"

if curl -sf http://localhost:11434/api/tags 2>/dev/null | grep -q "$OLLAMA_MODEL"; then
  ok "Model '${OLLAMA_MODEL}' is available"
else
  if $CHECK_ONLY; then
    warn "Model '${OLLAMA_MODEL}' not pulled"
    info "Pull: curl -s http://localhost:11434/api/pull -d '{\"name\":\"${OLLAMA_MODEL}\"}'"
  else
    if curl -sf http://localhost:11434/api/tags &>/dev/null; then
      info "Pulling model '${OLLAMA_MODEL}' (this may take a while)..."
      curl -s http://localhost:11434/api/pull -d "{\"name\":\"${OLLAMA_MODEL}\"}" | \
        while IFS= read -r line; do
          status=$(echo "$line" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
          if [ -n "$status" ]; then
            printf "\r  ${DIM}%s${RESET}                    " "$status"
          fi
        done
      echo ""
      if curl -sf http://localhost:11434/api/tags 2>/dev/null | grep -q "$OLLAMA_MODEL"; then
        ok "Model '${OLLAMA_MODEL}' pulled successfully"
      else
        fail "Model pull may have failed"
        ERRORS=$((ERRORS + 1))
      fi
    else
      warn "Ollama not running — cannot pull model"
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 8: Tests
# ─────────────────────────────────────────────────────────────────────────────
step "8/8 Tests"

if $CHECK_ONLY; then
  info "Skipping tests (check-only mode)"
else
  info "Running short tests (no Docker deps)..."
  if (cd "$SCRIPT_DIR" && go test -short ./... 2>&1 | tail -5); then
    ok "Short tests passed"
  else
    warn "Some tests failed (see output above)"
    ERRORS=$((ERRORS + 1))
  fi

  if docker ps &>/dev/null; then
    info "Running full tests (including Docker integration)..."
    if (cd "$SCRIPT_DIR" && go test ./... 2>&1 | tail -10); then
      ok "Full tests passed"
    else
      warn "Some integration tests failed"
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}━━━ Summary ━━━${RESET}"

if [ $ERRORS -eq 0 ]; then
  echo -e "${GREEN}${BOLD}  ✓ All checks passed!${RESET}"
else
  echo -e "${YELLOW}${BOLD}  ⚠ ${ERRORS} issue(s) need attention${RESET}"
fi

echo ""
echo -e "${DIM}Environment variables for dev-cli:${RESET}"
echo -e "  export DEV_CLI_OLLAMA_URL=http://localhost:11434"
echo -e "  export DEV_CLI_OLLAMA_MODEL=${OLLAMA_MODEL}"
echo ""
echo -e "${DIM}Quick test:${RESET}"
echo -e "  ./dev-cli doctor"
echo -e "  ./dev-cli ask --local \"hello world\""
echo ""

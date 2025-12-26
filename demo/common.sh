#!/bin/bash
# =============================================================================
# Common functions for demo scripts
# =============================================================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'
BOLD='\033[1m'
DIM='\033[2m'

DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${DEMO_DIR}/.."
CLI_BIN="${PROJECT_DIR}/dev-cli"

# Export temp directory for isolated testing
export DEV_CLI_LOG_DIR="${DEMO_DIR}/temp"
mkdir -p "$DEV_CLI_LOG_DIR"

demo_header() {
    local title="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${GREEN}$title${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

step() {
    local desc="$1"
    echo ""
    echo -e "${YELLOW}▸ ${desc}${NC}"
    echo ""
}

run_cmd() {
    local cmd="$1"
    echo -e "${DIM}$ ${cmd}${NC}"
    echo ""
    eval "$CLI_BIN ${cmd#dev-cli }" 2>&1 || true
    echo ""
}

run_shell() {
    local cmd="$1"
    echo -e "${DIM}$ ${cmd}${NC}"
    echo ""
    eval "$cmd" 2>&1 || true
    echo ""
}

wait_key() {
    echo -e "${CYAN}Press Enter to continue...${NC}"
    read -r
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

#!/bin/bash
# =============================================================================
# Demo: Doctor - System Health Checks
# Shows comprehensive health checks and auto-fix
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🏥 Doctor - System Health Checks"

echo -e "${WHITE}The 'doctor' command runs comprehensive health checks.${NC}"
echo -e "${WHITE}It can also auto-fix issues when possible.${NC}"
echo ""

step "Run all health checks"
run_cmd "doctor"

step "Output as JSON (for agent consumption)"
run_cmd "doctor --json"

step "Show only failures (quiet mode)"
run_cmd "doctor --quiet"

echo ""
info "Auto-fix mode: dev-cli doctor --fix"
echo -e "${WHITE}This will attempt to automatically fix any issues found, such as:${NC}"
echo -e "  • Starting Docker daemon"
echo -e "  • Creating missing directories"
echo -e "  • Starting Ollama container"
echo -e "  • Pulling required models"

echo ""
success "Doctor keeps your dev environment healthy!"

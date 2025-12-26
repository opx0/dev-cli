#!/bin/bash
# =============================================================================
# Demo: Explain/RCA - Root Cause Analysis
# Shows how to analyze failures and get fixes
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🔍 Explain/RCA - Root Cause Analysis"

echo -e "${WHITE}The 'explain' command analyzes your failed commands and suggests fixes.${NC}"
echo -e "${WHITE}Aliases: why, rca${NC}"
echo ""

step "First, let's simulate some failures for analysis"
run_cmd "log-event --command 'npm install' --exit-code 1 --cwd '/tmp/app' --output 'npm ERR! code ENOENT'"
run_cmd "log-event --command 'docker build .' --exit-code 1 --cwd '/tmp/app' --output 'Dockerfile not found'"
run_cmd "log-event --command 'git push origin main' --exit-code 128 --cwd '/tmp/app' --output 'fatal: Could not read from remote repository'"

step "Analyze the last failure (default)"
run_cmd "explain"

step "Analyze the last 3 failures"
run_cmd "explain --last 3"

step "Filter failures by command keyword"
run_cmd "explain --filter npm"

step "Filter by time (failures in last hour)"
run_cmd "explain --since 1h"

step "Use alias 'why' for quick analysis"
run_cmd "why --filter git"

echo ""
success "RCA helps you understand and fix errors faster!"

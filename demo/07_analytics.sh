#!/bin/bash
# =============================================================================
# Demo: Analytics - Proactive Debugging Insights
# Shows failure pattern analysis and suggestions
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "📊 Analytics - Proactive Debugging Insights"

echo -e "${WHITE}The 'analytics' command provides proactive debugging insights.${NC}"
echo -e "${WHITE}It analyzes your command history to identify patterns.${NC}"
echo ""

step "First, let's populate some command history for analysis"
# Simulate various commands with different results
run_cmd "log-event --command 'npm install' --exit-code 1 --cwd '/tmp/proj1' --output 'npm ERR!'"
run_cmd "log-event --command 'npm install' --exit-code 1 --cwd '/tmp/proj2' --output 'npm ERR! ENOENT'"
run_cmd "log-event --command 'npm install' --exit-code 0 --cwd '/tmp/proj3' --duration-ms 5000"
run_cmd "log-event --command 'go build' --exit-code 0 --cwd '/tmp/go1' --duration-ms 2000"
run_cmd "log-event --command 'go build' --exit-code 1 --cwd '/tmp/go2' --output 'undefined: main'"
run_cmd "log-event --command 'docker build .' --exit-code 1 --cwd '/tmp/docker' --output 'Dockerfile not found'"

step "Show failure patterns (default view)"
run_cmd "analytics"

step "Analyze a specific command pattern"
run_cmd "analytics --command npm"

step "Limit number of patterns shown"
run_cmd "analytics --limit 5"

step "Output as JSON for programmatic use"
run_cmd "analytics --json"

echo ""
success "Analytics helps you identify recurring issues!"
info "Aliases: stats, insights"

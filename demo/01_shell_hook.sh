#!/bin/bash
# =============================================================================
# Demo: Shell Hook Integration
# Shows how dev-cli captures commands automatically
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🔌 Shell Hook Integration"

echo -e "${WHITE}The shell hook automatically captures every command you run.${NC}"
echo -e "${WHITE}This enables intelligent analysis and suggestions.${NC}"
echo ""

step "View the shell hook script"
run_cmd "dev-cli init zsh | head -30"

step "The hook captures: command, exit code, duration, working directory, and output"
echo -e "${CYAN}When you add this to your ~/.zshrc:${NC}"
echo -e "${YELLOW}  eval \"\$(dev-cli init zsh)\"${NC}"
echo ""

step "Simulate logging a failed command"
run_cmd "dev-cli log-event --command 'npm install' --exit-code 1 --cwd '/tmp/project' --output 'npm ERR! ENOENT'"

step "Simulate logging a successful command"
run_cmd "dev-cli log-event --command 'go build' --exit-code 0 --cwd '/tmp/project' --duration-ms 1500"

echo ""
echo -e "${GREEN}✓ Commands are now tracked in SQLite for analysis!${NC}"

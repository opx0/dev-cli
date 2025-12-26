#!/bin/bash
# =============================================================================
# Demo: TUI - Interactive Dashboard
# Shows the full-featured terminal UI
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🖥️  TUI - Interactive Dashboard"

echo -e "${WHITE}The 'ui' command launches an interactive terminal dashboard.${NC}"
echo -e "${WHITE}It provides a complete mission control for your dev environment.${NC}"
echo ""

step "TUI Features"
echo -e "${CYAN}Tabs:${NC}"
echo -e "  • ${WHITE}Agent${NC} - Chat interface with AI, execute commands"
echo -e "  • ${WHITE}Containers${NC} - Docker container monitoring with live logs"
echo -e "  • ${WHITE}History${NC} - Command history browser with details"
echo ""

step "Key Bindings"
echo -e "  ${GREEN}1/2/3${NC}   - Switch tabs"
echo -e "  ${GREEN}Tab${NC}     - Cycle focus"
echo -e "  ${GREEN}i${NC}       - Insert mode (for typing)"
echo -e "  ${GREEN}Esc${NC}     - Normal mode"
echo -e "  ${GREEN}Ctrl+o${NC}  - Switch to OpenCode TUI"
echo -e "  ${GREEN}q${NC}       - Quit"
echo ""

step "Launch the TUI"
info "Command: dev-cli ui"
echo ""

echo -e "${YELLOW}Press Enter to launch the TUI (press 'q' to exit)...${NC}"
read -r

$CLI_BIN ui

echo ""
success "TUI provides a complete interactive experience!"

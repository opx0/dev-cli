#!/bin/bash
# =============================================================================
# Demo: Fix Agent - Autonomous Problem Solving
# Shows how the agent can fix issues autonomously
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🤖 Fix Agent - Autonomous Problem Solving"

echo -e "${WHITE}The 'fix' command uses AI to autonomously solve problems.${NC}"
echo -e "${WHITE}It proposes solutions and asks for permission before executing.${NC}"
echo ""

step "Fix asks AI to solve a task, then executes with approval"
info "Example: dev-cli fix 'create a hello world python script'"
echo ""

echo -e "${CYAN}How it works:${NC}"
echo -e "  1. You describe the task in natural language"
echo -e "  2. AI analyzes the problem and proposes a command"
echo -e "  3. You review and approve/reject the proposed fix"
echo -e "  4. If approved, the command is executed"
echo ""

step "Let's try a simple task"
info "Running: dev-cli fix 'print hello world'"
echo ""

# Note: This is interactive, so we just show the command
echo -e "${YELLOW}This command is interactive - it will ask:${NC}"
echo -e "${DIM}  I want to run: echo 'hello world'${NC}"
echo -e "${DIM}  Allow? [y/N]${NC}"
echo ""

# For non-interactive demo, show the concept
step "The agent can handle complex tasks like:"
echo -e "  • ${WHITE}dev-cli fix 'find and kill process on port 3000'${NC}"
echo -e "  • ${WHITE}dev-cli fix 'create a git branch for feature X'${NC}"
echo -e "  • ${WHITE}dev-cli fix 'compress all images in this folder'${NC}"
echo -e "  • ${WHITE}dev-cli fix 'set up a new npm project'${NC}"

echo ""
success "The fix agent is your autonomous coding assistant!"
warn "Always review proposed commands before approving!"

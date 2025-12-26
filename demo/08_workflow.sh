#!/bin/bash
# =============================================================================
# Demo: Workflow Automation
# Shows multi-step workflows with rollback and checkpointing
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🔄 Workflow Automation"

echo -e "${WHITE}The 'workflow' command executes multi-step automation.${NC}"
echo -e "${WHITE}Features: conditionals, rollback, checkpointing, resume.${NC}"
echo ""

step "View a sample workflow definition"
echo -e "${DIM}$ cat workflows/deploy_demo.yaml${NC}"
cat "${DEMO_DIR}/workflows/deploy_demo.yaml" 2>/dev/null || echo "Workflow file not found"
echo ""

step "Run a workflow"
run_cmd "workflow run ${DEMO_DIR}/workflows/deploy_demo.yaml"

step "List recent workflow runs"
run_cmd "workflow list"

step "Check status of a specific run"
info "dev-cli workflow status <run-id>"
echo ""

step "Resume a paused or failed workflow"
info "dev-cli workflow resume <run-id>"
echo ""

step "Manually trigger rollback"
info "dev-cli workflow rollback <run-id>"
echo ""

echo -e "${CYAN}Workflow Features:${NC}"
echo -e "  • ${WHITE}Sequential step execution${NC}"
echo -e "  • ${WHITE}Conditional branching (if/else)${NC}"
echo -e "  • ${WHITE}Automatic retry on failure${NC}"
echo -e "  • ${WHITE}Rollback on critical failures${NC}"
echo -e "  • ${WHITE}Checkpoint/resume for long operations${NC}"
echo -e "  • ${WHITE}Safe mode (preview before execute)${NC}"

echo ""
success "Workflows automate complex multi-step operations!"

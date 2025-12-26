#!/bin/bash
# =============================================================================
# Demo: Ask AI - Intelligent Research
# Shows how to query the AI for help
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "🧠 Ask AI - Intelligent Research"

echo -e "${WHITE}The 'ask' command queries AI (Ollama or Perplexity) for help.${NC}"
echo -e "${WHITE}Perfect for getting command suggestions and explanations.${NC}"
echo ""

step "Ask how to find large files"
run_cmd "ask how to find files larger than 100MB in linux"

step "Ask about Docker commands"
run_cmd "ask how to remove all stopped docker containers"

step "Ask with command count limit"
run_cmd "ask -n 5 how to check disk usage by folder"

step "You can also force local Ollama with --local flag"
info "dev-cli ask --local 'your question'"

echo ""
success "AI responses help you learn and work faster!"

#!/bin/bash
# =============================================================================
# dev-cli Demo Runner - Interactive Feature Showcase
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color
BOLD='\033[1m'

DEMO_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_BIN="${DEMO_DIR}/../dev-cli"

# Ensure CLI is built
if [[ ! -f "$CLI_BIN" ]]; then
    echo -e "${YELLOW}Building dev-cli...${NC}"
    cd "${DEMO_DIR}/.."
    go build -o dev-cli .
    cd "$DEMO_DIR"
fi

pause() {
    echo ""
    echo -e "${CYAN}Press Enter to continue...${NC}"
    read -r
}

header() {
    clear
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC}${BOLD}                    dev-cli Demo Suite                       ${PURPLE}║${NC}"
    echo -e "${PURPLE}║${NC}${WHITE}            The Autonomous Agentic Terminal                  ${PURPLE}║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

demo_header() {
    local title="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${GREEN}▶ $title${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

run_demo() {
    local script="$1"
    if [[ -f "${DEMO_DIR}/${script}" ]]; then
        bash "${DEMO_DIR}/${script}"
    else
        echo -e "${RED}Demo script not found: ${script}${NC}"
    fi
}

show_menu() {
    header
    echo -e "${WHITE}Select a demo to run:${NC}"
    echo ""
    echo -e "  ${GREEN}1)${NC} 🔌 Shell Hook Integration"
    echo -e "  ${GREEN}2)${NC} 🧠 Ask AI (Intelligent Research)"
    echo -e "  ${GREEN}3)${NC} 🔍 Explain/RCA (Root Cause Analysis)"
    echo -e "  ${GREEN}4)${NC} 👀 Watch Logs (Real-time Monitoring)"
    echo -e "  ${GREEN}5)${NC} 🤖 Fix Agent (Autonomous Fixes)"
    echo -e "  ${GREEN}6)${NC} 🏥 Doctor (Health Checks)"
    echo -e "  ${GREEN}7)${NC} 📊 Analytics (Proactive Insights)"
    echo -e "  ${GREEN}8)${NC} 🔄 Workflow Automation"
    echo -e "  ${GREEN}9)${NC} 🖥️  TUI (Interactive Dashboard)"
    echo -e "  ${GREEN}a)${NC} 🚀 Run ALL Demos"
    echo ""
    echo -e "  ${YELLOW}q)${NC} Quit"
    echo ""
    echo -ne "${CYAN}Enter choice: ${NC}"
}

run_all() {
    for i in 01 02 03 04 05 06 07 08; do
        run_demo "${i}_*.sh"
        pause
    done
}

main() {
    while true; do
        show_menu
        read -r choice
        case $choice in
            1) run_demo "01_shell_hook.sh"; pause ;;
            2) run_demo "02_ask_ai.sh"; pause ;;
            3) run_demo "03_explain_rca.sh"; pause ;;
            4) run_demo "04_watch_logs.sh"; pause ;;
            5) run_demo "05_fix_agent.sh"; pause ;;
            6) run_demo "06_doctor.sh"; pause ;;
            7) run_demo "07_analytics.sh"; pause ;;
            8) run_demo "08_workflow.sh"; pause ;;
            9) run_demo "09_tui.sh"; pause ;;
            a|A) run_all ;;
            q|Q) echo -e "${GREEN}Thanks for watching!${NC}"; exit 0 ;;
            *) echo -e "${RED}Invalid choice${NC}"; sleep 1 ;;
        esac
    done
}

main

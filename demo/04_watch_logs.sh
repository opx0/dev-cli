#!/bin/bash
# =============================================================================
# Demo: Watch Logs - Real-time Monitoring
# Shows how to monitor logs for errors
# =============================================================================

source "$(dirname "$0")/common.sh"

demo_header "👀 Watch Logs - Real-time Monitoring"

echo -e "${WHITE}The 'watch' command monitors logs in real-time.${NC}"
echo -e "${WHITE}When errors are detected, AI provides instant analysis.${NC}"
echo ""

# Create a sample log file
LOG_FILE="${DEMO_DIR}/temp/demo.log"
mkdir -p "${DEMO_DIR}/temp"
echo "Starting application..." > "$LOG_FILE"

step "Create a sample log file"
echo -e "${DIM}$ cat demo.log${NC}"
cat "$LOG_FILE"
echo ""

step "Start watching the log file (will run in background)"
info "Command: dev-cli watch --file ${LOG_FILE}"
echo ""

# Start watch in background
$CLI_BIN watch --file "$LOG_FILE" &
WATCH_PID=$!

sleep 1

step "Simulate writing log entries..."

echo "INFO: Server listening on port 3000" >> "$LOG_FILE"
echo -e "${DIM}[LOG] INFO: Server listening on port 3000${NC}"
sleep 1

echo "INFO: Processing request..." >> "$LOG_FILE"
echo -e "${DIM}[LOG] INFO: Processing request...${NC}"
sleep 1

step "Now let's simulate an error!"
echo "ERROR: Connection refused to database" >> "$LOG_FILE"
echo -e "${RED}[LOG] ERROR: Connection refused to database${NC}"

sleep 3

# Clean up
kill $WATCH_PID 2>/dev/null || true

echo ""
success "Watch command provides real-time AI analysis of errors!"
info "You can also watch Docker containers: dev-cli watch --docker container_name"

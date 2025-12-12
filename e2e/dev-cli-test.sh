#!/bin/bash
# Dev mode wrapper - logs to e2e/temp instead of ~/.devlogs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export DEV_CLI_LOG_DIR="$SCRIPT_DIR/temp"

# Run dev-cli with dev log path
exec "$SCRIPT_DIR/../dev-cli" "$@"

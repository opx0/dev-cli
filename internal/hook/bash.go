package hook

// BashHook is the bash shell integration script.
// Install with: eval "$(dev-cli init bash)"
//
// Uses a DEBUG trap to capture the command about to execute, plus
// PROMPT_COMMAND to record the exit code once it finishes. Interactive
// programs (vim, ssh, …) are skipped so we don't log UI sessions.
const BashHook = `# dev-cli Bash integration
# eval "$(dev-cli init bash)"

__DEVOPS_CMD=""
__DEVOPS_START_TIME=0
__DEVOPS_SKIP_LOG=0
__DEVOPS_LAST_OUTPUT=""

__DEVOPS_SKIP_CMDS=(vim vi nvim nano less more top htop man ssh tmux screen)

__devops_is_interactive() {
    local cmd_base="${1%% *}"
    for skip in "${__DEVOPS_SKIP_CMDS[@]}"; do
        [[ "$cmd_base" == "$skip" ]] && return 0
    done
    return 1
}

__devops_preexec() {
    # BASH_COMMAND is the command about to be executed. Skip our own hooks
    # (PROMPT_COMMAND reentry) and subshell housekeeping.
    [[ -n "$COMP_LINE" ]] && return        # during completion
    [[ "$BASH_COMMAND" == "__devops_precmd"* ]] && return
    [[ "$BASH_COMMAND" == "$PROMPT_COMMAND"* ]] && return
    __DEVOPS_CMD="$BASH_COMMAND"
    __DEVOPS_START_TIME=$(($(date +%s%N)/1000000))
    __DEVOPS_SKIP_LOG=0
    __DEVOPS_LAST_OUTPUT=""
    [[ "$__DEVOPS_CMD" == dcap\ * ]] && __DEVOPS_SKIP_LOG=1
    __devops_is_interactive "$__DEVOPS_CMD" && __DEVOPS_SKIP_LOG=1
}

__devops_suggest_fix() {
    local cmd="$1"
    local exit_code="$2"
    local output="$3"

    if [[ "$output" == *"Permission denied"* ]] || [[ "$output" == *"permission denied"* ]] || [[ $exit_code -eq 126 ]]; then
        echo -e "\033[33mTip:\033[0m Permission denied. Try: \033[1mdcap sudo !!\033[0m"
        return 0
    fi
    if [[ "$output" == *"command not found"* ]] || [[ $exit_code -eq 127 ]]; then
        local missing_cmd=$(echo "$output" | grep -oP "command not found: \K\w+" | head -1)
        if [[ -n "$missing_cmd" ]]; then
            echo -e "\033[33mTip:\033[0m Command '$missing_cmd' not found. Try: \033[1mdev-cli ask \"install $missing_cmd\"\033[0m"
        fi
        return 0
    fi
    if [[ "$output" == *"Cannot connect to the Docker daemon"* ]] || [[ "$output" == *"docker.sock"* ]]; then
        echo -e "\033[33mTip:\033[0m Docker not running. Try: \033[1msudo systemctl start docker\033[0m"
        return 0
    fi
    if [[ "$output" == *"not a git repository"* ]]; then
        echo -e "\033[33mTip:\033[0m Not a git repo. Try: \033[1mgit init\033[0m"
        return 0
    fi
    return 1
}

__devops_precmd() {
    local exit_code=$?
    [[ -z "$__DEVOPS_CMD" || $__DEVOPS_SKIP_LOG -eq 1 ]] && { __DEVOPS_CMD=""; return 0; }

    local end_time=$(($(date +%s%N)/1000000))
    local duration_ms=$((end_time - __DEVOPS_START_TIME))

    dev-cli log-event \
        --command "$__DEVOPS_CMD" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration_ms" >/dev/null 2>&1 &
    disown 2>/dev/null

    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        if ! __devops_suggest_fix "$__DEVOPS_CMD" "$exit_code" "$__DEVOPS_LAST_OUTPUT"; then
            echo -e "\033[90m× Failure logged. For AI analysis:\033[0m dcap \"$__DEVOPS_CMD\""
        fi
    fi
    __DEVOPS_CMD=""
    __DEVOPS_START_TIME=0
    __DEVOPS_SKIP_LOG=0
    __DEVOPS_LAST_OUTPUT=""
}

dcap() {
    local tmpfile=$(mktemp /tmp/devops_out.XXXXXX)
    local start=$(($(date +%s%N)/1000000))

    eval "$*" 2>&1 | tee "$tmpfile"
    local exit_code=${PIPESTATUS[0]}

    local end=$(($(date +%s%N)/1000000))
    local duration=$((end - start))
    local output=$(tail -c 10240 "$tmpfile" 2>/dev/null)
    __DEVOPS_LAST_OUTPUT="$output"

    dev-cli log-event \
        --command "$*" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration" \
        --output "$output" >/dev/null 2>&1

    rm -f "$tmpfile"

    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        __devops_suggest_fix "$*" "$exit_code" "$output"
        echo ""
        read -r -p $'\033[90mRun AI analysis? [y/N]:\033[0m ' response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            dev-cli explain --last 1 --interactive 2>/dev/null
        fi
    fi
    return $exit_code
}

trap '__devops_preexec' DEBUG
PROMPT_COMMAND="__devops_precmd; ${PROMPT_COMMAND}"
`

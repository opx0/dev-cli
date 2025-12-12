package hook

// ZshHook - shell integration for preexec/precmd capture
const ZshHook = `# dev-cli Zsh integration
# Source this file: eval "$(dev-cli hook zsh)"

# Store command info before execution
__devops_preexec() {
    __DEVOPS_CMD="$1"
    __DEVOPS_START_TIME=$(($(date +%s%N)/1000000))
}

# Log command after execution
__devops_precmd() {
    local exit_code=$?
    
    # Skip if no command was captured
    [[ -z "$__DEVOPS_CMD" ]] && return

    local end_time=$(($(date +%s%N)/1000000))
    local duration_ms=$((end_time - __DEVOPS_START_TIME))
    local cwd="$PWD"

    # Log the event
    dev-cli log-event \
        --command "$__DEVOPS_CMD" \
        --exit-code "$exit_code" \
        --cwd "$cwd" \
        --duration-ms "$duration_ms" 2>/dev/null

    # Show failure indicator
    if [[ $exit_code -ne 0 ]]; then
        echo "!! dev-cli captured failure (exit $exit_code)"
    fi

    # Reset for next command
    unset __DEVOPS_CMD
    unset __DEVOPS_START_TIME
}

# Hook into Zsh
autoload -Uz add-zsh-hook
add-zsh-hook preexec __devops_preexec
add-zsh-hook precmd __devops_precmd
`

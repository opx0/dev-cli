package hook

// ZshHook - shell integration for preexec/precmd capture with output logging
const ZshHook = `# dev-cli Zsh integration
# Source this file: eval "$(dev-cli hook zsh)"

# State variables
typeset -g __DEVOPS_CMD=""
typeset -g __DEVOPS_START_TIME=0
typeset -g __DEVOPS_SKIP_LOG=0

# Interactive commands to skip
typeset -ga __DEVOPS_SKIP_CMDS=(vim vi nvim nano less more top htop man ssh tmux screen)

# Check if command is interactive
__devops_is_interactive() {
    local cmd_base="${1%% *}"
    for skip in "${__DEVOPS_SKIP_CMDS[@]}"; do
        [[ "$cmd_base" == "$skip" ]] && return 0
    done
    return 1
}

# Called before command execution
__devops_preexec() {
    __DEVOPS_CMD="$1"
    __DEVOPS_START_TIME=$(($(date +%s%N)/1000000))
    __DEVOPS_SKIP_LOG=0
    
    # dcap handles its own logging - skip precmd logging
    [[ "$1" == dcap\ * ]] && __DEVOPS_SKIP_LOG=1
    
    # Skip interactive commands
    __devops_is_interactive "$1" && __DEVOPS_SKIP_LOG=1
}

# Called after command execution  
__devops_precmd() {
    local exit_code=$?
    
    # Skip if no command or already handled
    [[ -z "$__DEVOPS_CMD" || $__DEVOPS_SKIP_LOG -eq 1 ]] && return 0

    local end_time=$(($(date +%s%N)/1000000))
    local duration_ms=$((end_time - __DEVOPS_START_TIME))

    # Log the event (no output - precmd can't capture output)
    dev-cli log-event \
        --command "$__DEVOPS_CMD" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration_ms" 2>/dev/null &!

    # On failure (excluding Ctrl-C = 130), run RCA analysis
    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        dev-cli rca --interactive \
            --command "$__DEVOPS_CMD" \
            --exit-code "$exit_code" \
            --output "" 2>/dev/null
    fi

    # Reset state
    __DEVOPS_CMD=""
    __DEVOPS_START_TIME=0
    __DEVOPS_SKIP_LOG=0
}

# Wrapper function - use this for explicit output capture
# Usage: dcap "your command here"
dcap() {
    local tmpfile=$(mktemp /tmp/devops_out.XXXXXX)
    local start=$(($(date +%s%N)/1000000))
    
    # Run with tee to capture and display
    eval "$*" 2>&1 | tee "$tmpfile"
    local exit_code=${pipestatus[1]}
    
    local end=$(($(date +%s%N)/1000000))
    local duration=$((end - start))
    local output=$(tail -c 10240 "$tmpfile" 2>/dev/null)
    
    dev-cli log-event \
        --command "$*" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration" \
        --output "$output" 2>/dev/null
    
    rm -f "$tmpfile"
    
    # On failure (excluding Ctrl-C = 130), run RCA from log
    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        dev-cli rca --last 1 --interactive 2>/dev/null
    fi
    
    return $exit_code
}

# Hook into Zsh
autoload -Uz add-zsh-hook
add-zsh-hook preexec __devops_preexec
add-zsh-hook precmd __devops_precmd
`

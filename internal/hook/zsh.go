package hook

const ZshHook = `# dev-cli Zsh integration
# eval "$(dev-cli init zsh)"

typeset -g __DEVOPS_CMD=""
typeset -g __DEVOPS_START_TIME=0
typeset -g __DEVOPS_SKIP_LOG=0
typeset -ga __DEVOPS_SKIP_CMDS=(vim vi nvim nano less more top htop man ssh tmux screen)

__devops_is_interactive() {
    local cmd_base="${1%% *}"
    for skip in "${__DEVOPS_SKIP_CMDS[@]}"; do
        [[ "$cmd_base" == "$skip" ]] && return 0
    done
    return 1
}

__devops_preexec() {
    __DEVOPS_CMD="$1"
    __DEVOPS_START_TIME=$(($(date +%s%N)/1000000))
    __DEVOPS_SKIP_LOG=0
    [[ "$1" == dcap\ * ]] && __DEVOPS_SKIP_LOG=1
    __devops_is_interactive "$1" && __DEVOPS_SKIP_LOG=1
}

__devops_precmd() {
    local exit_code=$?
    [[ -z "$__DEVOPS_CMD" || $__DEVOPS_SKIP_LOG -eq 1 ]] && return 0

    local end_time=$(($(date +%s%N)/1000000))
    local duration_ms=$((end_time - __DEVOPS_START_TIME))

    dev-cli log-event \
        --command "$__DEVOPS_CMD" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration_ms" 2>/dev/null &!

    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        echo "\033[90m× Failure logged. For AI analysis:\033[0m dcap \"$__DEVOPS_CMD\""
    fi

    __DEVOPS_CMD=""
    __DEVOPS_START_TIME=0
    __DEVOPS_SKIP_LOG=0
}

dcap() {
    local tmpfile=$(mktemp /tmp/devops_out.XXXXXX)
    local start=$(($(date +%s%N)/1000000))
    
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
    
    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        dev-cli explain --last 1 --interactive 2>/dev/null
    fi
    
    return $exit_code
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __devops_preexec
add-zsh-hook precmd __devops_precmd
`

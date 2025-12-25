package hook

const ZshHook = `# dev-cli Zsh integration
# eval "$(dev-cli init zsh)"

typeset -g __DEVOPS_CMD=""
typeset -g __DEVOPS_START_TIME=0
typeset -g __DEVOPS_SKIP_LOG=0
typeset -g __DEVOPS_LAST_OUTPUT=""
typeset -g __DEVOPS_LAST_FAILURE_ID=""
typeset -g __DEVOPS_LAST_FAILURE_CMD=""
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
    __DEVOPS_LAST_OUTPUT=""
    [[ "$1" == dcap\ * ]] && __DEVOPS_SKIP_LOG=1
    __devops_is_interactive "$1" && __DEVOPS_SKIP_LOG=1
}

# Smart error detection and suggestions
__devops_suggest_fix() {
    local cmd="$1"
    local exit_code="$2"
    local output="$3"
    
    # Permission Denied - suggest sudo
    if [[ "$output" == *"Permission denied"* ]] || [[ "$output" == *"permission denied"* ]] || [[ $exit_code -eq 126 ]]; then
        echo "\033[33m💡 Tip:\033[0m Permission denied. Try: \033[1mdcap sudo !!\033[0m"
        return 0
    fi
    
    # Command not found - suggest install
    if [[ "$output" == *"command not found"* ]] || [[ $exit_code -eq 127 ]]; then
        local missing_cmd=$(echo "$output" | grep -oP "command not found: \K\w+" | head -1)
        if [[ -n "$missing_cmd" ]]; then
            echo "\033[33m💡 Tip:\033[0m Command '$missing_cmd' not found. Try: \033[1mdev-cli ask \"install $missing_cmd\"\033[0m"
        fi
        return 0
    fi
    
    # Docker not running
    if [[ "$output" == *"Cannot connect to the Docker daemon"* ]] || [[ "$output" == *"docker.sock"* ]]; then
        echo "\033[33m💡 Tip:\033[0m Docker not running. Try: \033[1msudo systemctl start docker\033[0m"
        return 0
    fi
    
    # Git not a repository
    if [[ "$output" == *"not a git repository"* ]]; then
        echo "\033[33m💡 Tip:\033[0m Not a git repo. Try: \033[1mgit init\033[0m"
        return 0
    fi
    
    # npm/node module not found
    if [[ "$output" == *"Cannot find module"* ]] || [[ "$output" == *"MODULE_NOT_FOUND"* ]]; then
        echo "\033[33m💡 Tip:\033[0m Missing module. Try: \033[1mnpm install\033[0m"
        return 0
    fi
    
    # Python module not found
    if [[ "$output" == *"ModuleNotFoundError"* ]] || [[ "$output" == *"No module named"* ]]; then
        local module=$(echo "$output" | grep -oP "No module named '\K[^']+")
        if [[ -n "$module" ]]; then
            echo "\033[33m💡 Tip:\033[0m Missing Python module. Try: \033[1mpip install $module\033[0m"
        fi
        return 0
    fi
    
    # Port already in use
    if [[ "$output" == *"address already in use"* ]] || [[ "$output" == *"EADDRINUSE"* ]]; then
        echo "\033[33m💡 Tip:\033[0m Port in use. Find process: \033[1mlsof -i :<port> | grep LISTEN\033[0m"
        return 0
    fi
    
    return 1
}

# Check for unresolved failure and prompt user
__devops_check_resolution() {
    # Check if there's an unresolved failure
    local failure_info=$(dev-cli check-last-failure 2>/dev/null)
    if [[ -z "$failure_info" ]]; then
        __DEVOPS_LAST_FAILURE_ID=""
        __DEVOPS_LAST_FAILURE_CMD=""
        return
    fi
    
    __DEVOPS_LAST_FAILURE_ID="${failure_info%%|*}"
    __DEVOPS_LAST_FAILURE_CMD="${failure_info#*|}"
}

__devops_prompt_resolution() {
    local failure_id="$1"
    local failure_cmd="$2"
    
    echo ""
    echo "\033[32m✓\033[0m Success after failure: \033[90m$failure_cmd\033[0m"
    echo -n "\033[33m❓ Did this fix the issue? [y/n/skip]: \033[0m"
    read -r response
    
    case "$response" in
        [Yy]*)
            dev-cli mark-resolved --id "$failure_id" --resolution solution 2>/dev/null
            echo "\033[32m✓\033[0m Marked as solution!"
            ;;
        [Nn]*)
            dev-cli mark-resolved --id "$failure_id" --resolution unrelated 2>/dev/null
            echo "\033[90m○ Marked as unrelated\033[0m"
            ;;
        *)
            dev-cli mark-resolved --id "$failure_id" --resolution skipped 2>/dev/null
            echo "\033[90m○ Skipped\033[0m"
            ;;
    esac
    
    __DEVOPS_LAST_FAILURE_ID=""
    __DEVOPS_LAST_FAILURE_CMD=""
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
        # Command failed - check for unresolved failure after a short delay
        (sleep 0.1 && __devops_check_resolution) &!
        
        # Try smart suggestion first
        if ! __devops_suggest_fix "$__DEVOPS_CMD" "$exit_code" "$__DEVOPS_LAST_OUTPUT"; then
            # Fallback to generic message
            echo "\033[90m× Failure logged. For AI analysis:\033[0m dcap \"$__DEVOPS_CMD\""
        fi
    elif [[ $exit_code -eq 0 && -n "$__DEVOPS_LAST_FAILURE_ID" ]]; then
        # Command succeeded and there was a prior unresolved failure
        __devops_prompt_resolution "$__DEVOPS_LAST_FAILURE_ID" "$__DEVOPS_LAST_FAILURE_CMD"
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
    local exit_code=${pipestatus[1]}
    
    local end=$(($(date +%s%N)/1000000))
    local duration=$((end - start))
    local output=$(tail -c 10240 "$tmpfile" 2>/dev/null)
    __DEVOPS_LAST_OUTPUT="$output"
    
    dev-cli log-event \
        --command "$*" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration-ms "$duration" \
        --output "$output" 2>/dev/null
    
    rm -f "$tmpfile"
    
    if [[ $exit_code -ne 0 && $exit_code -ne 130 ]]; then
        # Check for unresolved failure after logging
        sleep 0.1
        __devops_check_resolution
        
        # Show smart suggestion
        __devops_suggest_fix "$*" "$exit_code" "$output"
        echo ""
        # Offer AI analysis
        echo -n "\033[90mRun AI analysis? [y/N]:\033[0m "
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            dev-cli explain --last 1 --interactive 2>/dev/null
        fi
    elif [[ $exit_code -eq 0 && -n "$__DEVOPS_LAST_FAILURE_ID" ]]; then
        # Success after failure - prompt for resolution
        __devops_prompt_resolution "$__DEVOPS_LAST_FAILURE_ID" "$__DEVOPS_LAST_FAILURE_CMD"
    fi
    
    return $exit_code
}

# Initialize by checking for any pending unresolved failures
__devops_check_resolution

autoload -Uz add-zsh-hook
add-zsh-hook preexec __devops_preexec
add-zsh-hook precmd __devops_precmd
`

#!/bin/bash

# Bash completion for kubectl oadp plugin that integrates with kubectl's completion system
# This script properly handles "kubectl oadp" and "oc oadp" commands

# First, ensure kubectl/oc completion is loaded
if ! complete -p kubectl &>/dev/null; then
    if command -v kubectl &>/dev/null; then
        source <(kubectl completion bash 2>/dev/null)
    fi
fi

if ! complete -p oc &>/dev/null; then
    # Check for oc in common locations, prioritizing ~/.local/bin
    oc_cmd=""
    if [[ -x "$HOME/.local/bin/oc" ]]; then
        oc_cmd="$HOME/.local/bin/oc"
    elif command -v oc &>/dev/null; then
        oc_cmd="oc"
    elif [[ -x "/usr/local/bin/oc" ]]; then
        oc_cmd="/usr/local/bin/oc"
    fi
    
    if [[ -n "$oc_cmd" ]]; then
        # Load oc completion and ensure it gets registered
        eval "$($oc_cmd completion bash 2>/dev/null)"
        echo "Loaded oc completion from: $oc_cmd"
    fi
fi

# Override the kubectl completion function to handle oadp plugin
_kubectl_oadp_plugin() {
    local cur prev words cword
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    words=("${COMP_WORDS[@]}")
    cword=$COMP_CWORD

    # Find the position of "oadp" in the command line
    local oadp_index=-1
    for ((i=0; i<${#words[@]}; i++)); do
        if [[ "${words[i]}" == "oadp" ]]; then
            oadp_index=$i
            break
        fi
    done

    # If "oadp" is not found, fall back to default kubectl completion
    if [[ $oadp_index -eq -1 ]]; then
        return 1
    fi

    # Calculate position relative to "oadp"
    local pos=$((cword - oadp_index))

    case $pos in
        1)
            # Completing after "kubectl/oc oadp"
            local commands="backup restore version client nabsl-request nonadmin na"
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            return 0
            ;;
        2)
            # Completing after "kubectl/oc oadp SUBCOMMAND"
            case "${words[$((oadp_index + 1))]}" in
                backup)
                    COMPREPLY=($(compgen -W "create delete describe download get logs" -- "$cur"))
                    ;;
                restore)
                    COMPREPLY=($(compgen -W "create delete describe get logs" -- "$cur"))
                    ;;
                client)
                    COMPREPLY=($(compgen -W "config" -- "$cur"))
                    ;;
                nabsl-request)
                    COMPREPLY=($(compgen -W "get describe approve reject" -- "$cur"))
                    ;;
                nonadmin|na)
                    COMPREPLY=($(compgen -W "backup bsl" -- "$cur"))
                    ;;
            esac
            return 0
            ;;
        3)
            # Completing after "kubectl/oc oadp nonadmin/na SUBCOMMAND"
            if [[ "${words[$((oadp_index + 1))]}" == "nonadmin" || "${words[$((oadp_index + 1))]}" == "na" ]]; then
                case "${words[$((oadp_index + 2))]}" in
                    backup)
                        COMPREPLY=($(compgen -W "create get logs describe delete" -- "$cur"))
                        ;;
                    bsl)
                        COMPREPLY=($(compgen -W "create" -- "$cur"))
                        ;;
                esac
            fi
            return 0
            ;;
        *)
            # For flags/options
            if [[ "$cur" == -* ]]; then
                local flags="--help -h --namespace -n --kubeconfig --context --include-resources --exclude-resources --include-namespaces --exclude-namespaces --snapshot-volumes --wait --dry-run -o --output"
                COMPREPLY=($(compgen -W "$flags" -- "$cur"))
                return 0
            fi
            ;;
    esac

    return 1
}

# Store the original kubectl completion function
if declare -f _kubectl &>/dev/null; then
    _kubectl_original() {
        # Call the original kubectl completion
        _kubectl "$@"
    }
fi

if declare -f _oc &>/dev/null; then
    _oc_original() {
        # Call the original oc completion
        _oc "$@"
    }
fi

# New kubectl completion function that handles oadp plugin
_kubectl_with_oadp() {
    # Check if this is an oadp command
    local words=("${COMP_WORDS[@]}")
    for word in "${words[@]}"; do
        if [[ "$word" == "oadp" ]]; then
            _kubectl_oadp_plugin
            if [[ ${#COMPREPLY[@]} -gt 0 ]]; then
                return 0
            fi
            break
        fi
    done

    # Fall back to original kubectl completion
    if declare -f _kubectl_original &>/dev/null; then
        _kubectl_original
    fi
}

# New oc completion function that handles oadp plugin
_oc_with_oadp() {
    # Check if this is an oadp command
    local words=("${COMP_WORDS[@]}")
    for word in "${words[@]}"; do
        if [[ "$word" == "oadp" ]]; then
            _kubectl_oadp_plugin
            if [[ ${#COMPREPLY[@]} -gt 0 ]]; then
                return 0
            fi
            break
        fi
    done

    # Fall back to original oc completion
    if declare -f _oc_original &>/dev/null; then
        _oc_original
    fi
}

# Override the completion functions
complete -F _kubectl_with_oadp kubectl
complete -F _oc_with_oadp oc

# Test function
test_kubectl_oadp_completion() {
    echo "=== Testing kubectl oadp completion ==="
    
    echo "Test 1: kubectl oadp <TAB>"
    COMP_WORDS=("kubectl" "oadp" "")
    COMP_CWORD=2
    _kubectl_oadp_plugin
    echo "Results: ${COMPREPLY[*]}"
    echo
    
    echo "Test 2: oc oadp nonadmin <TAB>"
    COMP_WORDS=("oc" "oadp" "nonadmin" "")
    COMP_CWORD=3
    _kubectl_oadp_plugin
    echo "Results: ${COMPREPLY[*]}"
    echo
    
    echo "Test 3: kubectl oadp na backup <TAB>"
    COMP_WORDS=("kubectl" "oadp" "na" "backup" "")
    COMP_CWORD=4
    _kubectl_oadp_plugin
    echo "Results: ${COMPREPLY[*]}"
    echo
}

echo "kubectl/oc oadp completion loaded!"
echo "Try: kubectl oadp <TAB> or oc oadp <TAB>"
echo "Run 'test_kubectl_oadp_completion' to test"
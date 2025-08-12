#!/bin/bash

# Bash completion script for kubectl oadp plugin
# Source this file or add it to your bash completion directory

_kubectl_oadp_complete() {
    local cur prev words cword
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    words=("${COMP_WORDS[@]}")
    cword=$COMP_CWORD

    # Find where 'oadp' appears in the command line
    local oadp_index=0
    for ((i=0; i<${#words[@]}; i++)); do
        if [[ "${words[i]}" == "oadp" ]]; then
            oadp_index=$i
            break
        fi
    done

    # Calculate position relative to 'oadp' command
    local pos=$((cword - oadp_index))

    case $pos in
        1)
            # First level subcommands after 'oadp'
            COMPREPLY=($(compgen -W "backup restore version client nabsl-request nonadmin na" -- "$cur"))
            ;;
        2)
            # Second level - depends on first subcommand
            local subcmd="${words[$((oadp_index + 1))]}"
            case "$subcmd" in
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
                version)
                    # No subcommands for version
                    COMPREPLY=()
                    ;;
            esac
            ;;
        3)
            # Third level - for nonadmin commands
            local subcmd="${words[$((oadp_index + 1))]}"
            local subsubcmd="${words[$((oadp_index + 2))]}"
            case "$subcmd" in
                nonadmin|na)
                    case "$subsubcmd" in
                        backup)
                            COMPREPLY=($(compgen -W "create get logs describe delete" -- "$cur"))
                            ;;
                        bsl)
                            COMPREPLY=($(compgen -W "create" -- "$cur"))
                            ;;
                    esac
                    ;;
            esac
            ;;
        *)
            # For resource names and flags, provide common options
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--help -h --namespace -n --kubeconfig --context --include-resources --exclude-resources --include-namespaces --exclude-namespaces --snapshot-volumes --wait --dry-run -o --output" -- "$cur"))
            fi
            ;;
    esac
}

# Register completion for different command forms
complete -F _kubectl_oadp_complete kubectl-oadp
complete -F _kubectl_oadp_complete oadp

# For kubectl plugin integration
_kubectl_oadp() {
    _kubectl_oadp_complete
}

# Register with kubectl if available
if command -v kubectl >/dev/null 2>&1; then
    complete -F _kubectl_oadp_complete kubectl
fi
#!/bin/bash

# Bash completion for standalone oadp binary (kubectl-oadp)

_oadp_complete() {
    local cur prev words cword
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Get all words in the command line
    words=("${COMP_WORDS[@]}")
    cword=$COMP_CWORD

    # Handle 'oadp ...' or 'kubectl-oadp ...' command structure
    case $cword in
        1)
            # Completing after 'oadp' or 'kubectl-oadp'
            local commands="backup restore version client nabsl-request nonadmin na"
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            ;;
        2)
            # Completing after 'oadp SUBCOMMAND'
            case "${words[1]}" in
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
            ;;
        3)
            # Completing after 'oadp nonadmin/na SUBCOMMAND'
            if [[ "${words[1]}" == "nonadmin" || "${words[1]}" == "na" ]]; then
                case "${words[2]}" in
                    backup)
                        COMPREPLY=($(compgen -W "create get logs describe delete" -- "$cur"))
                        ;;
                    bsl)
                        COMPREPLY=($(compgen -W "create" -- "$cur"))
                        ;;
                esac
            fi
            ;;
        *)
            # For flags/options
            if [[ "$cur" == -* ]]; then
                local flags="--help -h --namespace -n --kubeconfig --context --include-resources --exclude-resources --include-namespaces --exclude-namespaces --snapshot-volumes --wait --dry-run -o --output"
                COMPREPLY=($(compgen -W "$flags" -- "$cur"))
            fi
            ;;
    esac
}

# Register completion for both binary names
complete -F _oadp_complete oadp
complete -F _oadp_complete kubectl-oadp

echo "OADP standalone binary completion loaded!"
echo "Try typing: oadp <TAB> or kubectl-oadp <TAB>"
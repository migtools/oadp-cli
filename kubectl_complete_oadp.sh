#!/bin/bash

# Bash completion for kubectl oadp plugin

_kubectl_oadp() {
    local cur prev words cword
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Get all words in the command line
    words=("${COMP_WORDS[@]}")
    cword=$COMP_CWORD

    # Handle 'kubectl oadp ...' command structure
    case $cword in
        2)
            # Completing after 'kubectl oadp'
            local commands="backup restore version client nabsl-request nonadmin na"
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            ;;
        3)
            # Completing after 'kubectl oadp SUBCOMMAND'
            case "${words[2]}" in
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
        4)
            # Completing after 'kubectl oadp nonadmin/na SUBCOMMAND'
            if [[ "${words[2]}" == "nonadmin" || "${words[2]}" == "na" ]]; then
                case "${words[3]}" in
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

# Simple test function
test_oadp_completion() {
    echo "=== Testing OADP Completion ==="
    
    # Test 1: Basic completion
    echo "Test: kubectl oadp <TAB>"
    COMP_WORDS=("kubectl" "oadp" "")
    COMP_CWORD=2
    _kubectl_oadp
    echo "Results: ${COMPREPLY[*]}"
    echo
    
    # Test 2: Nonadmin completion
    echo "Test: kubectl oadp nonadmin <TAB>"
    COMP_WORDS=("kubectl" "oadp" "nonadmin" "")
    COMP_CWORD=3
    _kubectl_oadp
    echo "Results: ${COMPREPLY[*]}"
    echo
    
    # Test 3: Backup completion
    echo "Test: kubectl oadp nonadmin backup <TAB>"
    COMP_WORDS=("kubectl" "oadp" "nonadmin" "backup" "")
    COMP_CWORD=4
    _kubectl_oadp
    echo "Results: ${COMPREPLY[*]}"
    echo
}

# Register completion for kubectl
complete -F _kubectl_oadp kubectl

echo "OADP bash completion loaded successfully!"
echo "Try typing: kubectl oadp <TAB>"
echo "Run 'test_oadp_completion' to verify it works"
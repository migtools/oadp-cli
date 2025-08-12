#!/bin/bash

# Installation script for OADP bash completion

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPLETION_SCRIPT="$SCRIPT_DIR/kubectl_oadp_completion.sh"

echo "Installing OADP bash completion..."

# Check if the completion script exists
if [[ ! -f "$COMPLETION_SCRIPT" ]]; then
    echo "Error: Completion script not found at $COMPLETION_SCRIPT"
    exit 1
fi

# Option 1: Install to user's bash completion directory
USER_COMPLETION_DIR="$HOME/.bash_completion.d"
if [[ ! -d "$USER_COMPLETION_DIR" ]]; then
    mkdir -p "$USER_COMPLETION_DIR"
fi

cp "$COMPLETION_SCRIPT" "$USER_COMPLETION_DIR/kubectl_oadp"
echo "Installed to: $USER_COMPLETION_DIR/kubectl_oadp"

# Option 2: Add to .bashrc if not already there
BASHRC="$HOME/.bashrc"
SOURCE_LINE="source $USER_COMPLETION_DIR/kubectl_oadp"

if [[ -f "$BASHRC" ]] && ! grep -q "kubectl_oadp" "$BASHRC"; then
    echo "" >> "$BASHRC"
    echo "# OADP kubectl plugin completion" >> "$BASHRC"
    echo "$SOURCE_LINE" >> "$BASHRC"
    echo "Added source line to $BASHRC"
fi

echo ""
echo "Installation complete!"
echo ""
echo "To activate completion in your current shell, run:"
echo "  source $USER_COMPLETION_DIR/kubectl_oadp"
echo ""
echo "Or restart your terminal to load it automatically."
echo ""
echo "Test it with:"
echo "  kubectl oadp <TAB>"
echo "  oc oadp <TAB>"
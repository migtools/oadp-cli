# OADP CLI Tab Completion Setup

This document explains how to set up tab completion for the `oadp` cli plugin.

## Overview

Setting up completion requires 3 steps:

1. **Prerequisites**: Ensure kubectl/oc completion is working
2. **Create wrapper script**: `kubectl_complete-oadp` (and optionally `oc_complete-oadp`) - these names follow kubectl/oc plugin completion conventions for automatic discovery

3. **Configure shell**: Add completion config to `.zshrc` or `.bashrc`

**Quick test**: After setup, `kubectl oadp <TAB><TAB>` should show available commands.

## Prerequisites

Before setting up kubectl-oadp completion, you need:

1. **kubectl completion** configured in your shell
2. **oc completion** configured in your shell (if using OpenShift)

### Check if kubectl completion is working:
```bash
kubectl get <TAB><TAB>
# Should show: pods, services, deployments, etc.
```

### Set up kubectl completion if missing:

**For zsh** (add to `~/.zshrc`):
```bash
if command -v kubectl >/dev/null 2>&1; then
  source <(kubectl completion zsh)
  compdef _kubectl kubectl
fi
```

**For bash** (add to `~/.bashrc`):
```bash
if command -v kubectl >/dev/null 2>&1; then
  source <(kubectl completion bash)
fi
```

### Set up oc completion if using OpenShift:

**For zsh** (add to `~/.zshrc`):
```bash
if command -v oc >/dev/null 2>&1; then
  source <(oc completion zsh)
  compdef _oc oc
fi
```

**For bash** (add to `~/.bashrc`):
```bash
if command -v oc >/dev/null 2>&1; then
  source <(oc completion bash)
fi
```

## What You Need

Tab completion requires two components:
1. **Completion wrapper script** (`kubectl_complete-oadp`)
2. **Shell configuration** (in `.zshrc` or `.bashrc`)

## Quick Setup

### 1. Install the Completion Wrapper

Create the wrapper script in the same directory as your `kubectl-oadp` binary:

#### For kubectl plugin completion:

```bash
# If kubectl-oadp is in ~/.local/bin (most common)
cat > ~/.local/bin/kubectl_complete-oadp << 'EOF'
#!/bin/bash
# Wrapper script for kubectl plugin completion
exec ~/.local/bin/kubectl-oadp __complete "$@"
EOF
chmod +x ~/.local/bin/kubectl_complete-oadp
```

#### For oc plugin completion:

If you also want `oc oadp` completion (using the same binary as an oc plugin):

```bash
# Create oc completion wrapper
cat > ~/.local/bin/oc_complete-oadp << 'EOF'
#!/bin/bash
# Wrapper script for oc plugin completion
exec ~/.local/bin/kubectl-oadp __complete "$@"
EOF
chmod +x ~/.local/bin/oc_complete-oadp
```

**For other locations:**
- If using `~/bin`: Replace `~/.local/bin` with `~/bin`
- If using `/usr/local/bin`: Replace `~/.local/bin` with `/usr/local/bin`

### 2. Configure Your Shell

Add completion configuration to your shell's rc file:

**For zsh** (add to `~/.zshrc`):
```bash
# kubectl-oadp completion
if [ -f "$HOME/.local/bin/kubectl-oadp" ]; then
  source <($HOME/.local/bin/kubectl-oadp completion zsh)
  compdef _oadp kubectl-oadp
fi
```

**For bash** (add to `~/.bashrc`):
```bash
# kubectl-oadp completion
if [ -f "$HOME/.local/bin/kubectl-oadp" ]; then
  source <($HOME/.local/bin/kubectl-oadp completion bash)
fi
```

### 3. Reload Your Shell

```bash
# For zsh
source ~/.zshrc

# For bash  
source ~/.bashrc
```

## Test It Works

Try typing and pressing TAB twice:
```bash
kubectl oadp <TAB><TAB>
```

You should see available commands like `backup`, `nonadmin`, `nabsl`, etc.

## How It Works

1. **You type**: `kubectl oadp backup <TAB>`
2. **Shell detects**: Tab completion request for kubectl plugin
3. **Shell calls**: `kubectl_complete-oadp __complete kubectl oadp backup`
4. **Wrapper forwards to**: `kubectl-oadp __complete kubectl oadp backup`
5. **Plugin returns**: Available completions (create, delete, get, etc.)
6. **Shell shows**: The completion options

## Troubleshooting

### Completion Not Working?

**Check if wrapper exists:**
```bash
ls -la ~/.local/bin/kubectl_complete-oadp
```

**Check if it's executable:**
```bash
chmod +x ~/.local/bin/kubectl_complete-oadp
```

**Test wrapper directly:**
```bash
kubectl_complete-oadp __complete kubectl oadp
```

**Check shell configuration:**
```bash
# For zsh
grep -A5 "kubectl-oadp completion" ~/.zshrc

# For bash
grep -A3 "kubectl-oadp completion" ~/.bashrc
```

### Path Issues?

Make sure both files are in the same directory and that directory is in your PATH:
```bash
echo $PATH | grep -o '[^:]*\.local/bin[^:]*'
which kubectl-oadp
which kubectl_complete-oadp
which oc_complete-oadp  # if using oc completion
```

## Uninstalling Completion

### Remove the Wrapper Scripts
```bash
# Remove kubectl completion wrapper
rm ~/.local/bin/kubectl_complete-oadp

# Remove oc completion wrapper (if created)
rm ~/.local/bin/oc_complete-oadp
```

### Remove Shell Configuration

**For zsh:**
1. Edit `~/.zshrc`
2. Remove the block starting with `# kubectl-oadp completion` through the `fi` line
3. Run `source ~/.zshrc`

**For bash:**
1. Edit `~/.bashrc` 
2. Remove the block starting with `# kubectl-oadp completion` through the `fi` line
3. Run `source ~/.bashrc`

## Advanced: Custom Locations

If your `kubectl-oadp` binary is in a non-standard location, update the wrapper script paths:

```bash
# Example for custom location /opt/oadp/bin
cat > /opt/oadp/bin/kubectl_complete-oadp << 'EOF'
#!/bin/bash
exec /opt/oadp/bin/kubectl-oadp __complete "$@"
EOF

# Update shell config accordingly
if [ -f "/opt/oadp/bin/kubectl-oadp" ]; then
  source <(/opt/oadp/bin/kubectl-oadp completion zsh)
  compdef _oadp kubectl-oadp
fi
```

## Why This Design?

- **kubectl convention**: Follows standard kubectl plugin completion patterns
- **Automatic discovery**: Shell finds completion scripts by naming convention
- **Separation**: Keeps completion logic separate from main plugin
- **Flexibility**: Works with any installation location

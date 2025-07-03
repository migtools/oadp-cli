# Makefile for OADP CLI
# 
# This Makefile provides convenient targets for building, testing, and installing
# the OADP CLI as a kubectl plugin.

# Variables
BINARY_NAME = kubectl-oadp
LOCAL_BINARY = oadp
INSTALL_PATH = /usr/local/bin
GO_FILES = $(shell find . -name '*.go' -not -path './vendor/*')

# Default target
.PHONY: help
help: ## Show this help message
	@echo "OADP CLI Makefile"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
.PHONY: build
build: ## Build the CLI binary for local development
	@echo "Building $(LOCAL_BINARY) for local development..."
	go build -o $(LOCAL_BINARY) .
	@echo "✅ Built $(LOCAL_BINARY) successfully!"

.PHONY: build-plugin
build-plugin: ## Build the kubectl plugin binary
	@echo "Building $(BINARY_NAME) kubectl plugin..."
	go build -o $(BINARY_NAME) .
	@echo "✅ Built $(BINARY_NAME) successfully!"

# Installation targets
.PHONY: install
install: build-plugin ## Build and install the kubectl plugin (requires sudo)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✅ $(BINARY_NAME) plugin installed successfully!"
	@echo "You can now use: kubectl oadp --help"

.PHONY: install-local
install-local: build-plugin ## Install the kubectl plugin to ~/bin (no sudo required)
	@echo "Installing $(BINARY_NAME) to ~/bin..."
	@mkdir -p ~/bin
	mv $(BINARY_NAME) ~/bin/
	@echo "✅ $(BINARY_NAME) plugin installed to ~/bin!"
	@echo "Make sure ~/bin is in your PATH"
	@echo "You can now use: kubectl oadp --help"

# Testing targets
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	go test ./...
	@echo "✅ All tests passed!"

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -cover ./...

# Development targets
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "✅ Code formatted!"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "✅ Vet checks passed!"

.PHONY: mod-tidy
mod-tidy: ## Tidy up go.mod
	@echo "Tidying go.mod..."
	go mod tidy
	@echo "✅ Dependencies tidied!"

.PHONY: check
check: fmt vet test ## Run formatting, vetting, and tests

# Cleanup targets
.PHONY: clean
clean: ## Remove built binaries
	@echo "Cleaning up built binaries..."
	@rm -f $(BINARY_NAME) $(LOCAL_BINARY)
	@echo "✅ Cleanup complete!"

.PHONY: uninstall
uninstall: ## Remove the installed kubectl plugin
	@echo "Removing $(BINARY_NAME) from $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) plugin uninstalled!"

.PHONY: uninstall-local
uninstall-local: ## Remove the kubectl plugin from ~/bin
	@echo "Removing $(BINARY_NAME) from ~/bin..."
	@rm -f ~/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) plugin uninstalled from ~/bin!"

# Utility targets
.PHONY: verify-install
verify-install: ## Verify the kubectl plugin is installed and working
	@echo "Verifying kubectl plugin installation..."
	@if command -v kubectl oadp >/dev/null 2>&1; then \
		echo "✅ kubectl oadp plugin is installed and accessible!"; \
		kubectl oadp --help | head -5; \
	else \
		echo "❌ kubectl oadp plugin not found in PATH"; \
		exit 1; \
	fi

.PHONY: dev-setup
dev-setup: mod-tidy fmt vet build ## Set up development environment
	@echo "✅ Development environment ready!"

# File watching (requires 'entr' tool)
.PHONY: watch
watch: ## Watch for changes and rebuild (requires 'entr' tool)
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for changes... (press Ctrl+C to stop)"; \
		find . -name '*.go' | entr -r make build; \
	else \
		echo "❌ 'entr' tool not found. Install with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
		exit 1; \
	fi

# Show current status
.PHONY: status
status: ## Show build status and installed version
	@echo "=== OADP CLI Status ==="
	@echo ""
	@echo "📁 Repository:"
	@pwd
	@echo ""
	@echo "🔧 Local binaries:"
	@ls -la $(LOCAL_BINARY) $(BINARY_NAME) 2>/dev/null || echo "  No local binaries found"
	@echo ""
	@echo "📦 Installed plugin:"
	@ls -la $(INSTALL_PATH)/$(BINARY_NAME) 2>/dev/null || echo "  Plugin not installed in $(INSTALL_PATH)"
	@ls -la ~/bin/$(BINARY_NAME) 2>/dev/null || echo "  Plugin not installed in ~/bin"
	@echo ""
	@echo "✅ Plugin accessibility:"
	@if command -v kubectl oadp >/dev/null 2>&1; then \
		echo "  kubectl oadp is accessible"; \
	else \
		echo "  kubectl oadp is NOT accessible"; \
	fi 
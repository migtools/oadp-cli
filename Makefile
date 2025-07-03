# Makefile for OADP CLI
# 
# Simple Makefile for building, testing, and installing the OADP CLI

# Variables
BINARY_NAME = kubectl-oadp
INSTALL_PATH = /usr/local/bin

# Default target
.PHONY: help
help: ## Show this help message
	@echo "OADP CLI Makefile"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
.PHONY: build
build: ## Build the kubectl plugin binary
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .
	@echo "✅ Built $(BINARY_NAME) successfully!"

# Installation targets
.PHONY: install
install: build ## Build and install the kubectl plugin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✅ $(BINARY_NAME) installed successfully!"
	@echo "You can now use: kubectl oadp --help"

# Testing targets
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	go test ./...
	@echo "✅ Tests completed!"

# Cleanup targets
.PHONY: clean
clean: ## Remove built binaries
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "✅ Cleanup complete!"

# Status and utility targets
.PHONY: status
status: ## Show build status and installation info
	@echo "=== OADP CLI Status ==="
	@echo ""
	@echo "📁 Repository:"
	@pwd
	@echo ""
	@echo "🔧 Local binary:"
	@ls -la $(BINARY_NAME) 2>/dev/null || echo "  No local binary found"
	@echo ""
	@echo "📦 Installed plugin:"
	@ls -la $(INSTALL_PATH)/$(BINARY_NAME) 2>/dev/null || echo "  Plugin not installed"
	@echo ""
	@echo "✅ Plugin accessibility:"
	@if command -v kubectl oadp >/dev/null 2>&1; then \
		echo "  kubectl oadp is accessible"; \
	else \
		echo "  kubectl oadp is NOT accessible"; \
	fi 

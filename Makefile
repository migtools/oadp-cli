# Makefile for OADP CLI
# 
# Simple Makefile for building, testing, and installing the OADP CLI

# Variables
BINARY_NAME = kubectl-oadp
INSTALL_PATH ?= /usr/local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Platform variables for multi-arch builds
# Usage: make build PLATFORM=linux/amd64
PLATFORM ?= 
GOOS = $(word 1,$(subst /, ,$(PLATFORM)))
GOARCH = $(word 2,$(subst /, ,$(PLATFORM)))

# Default target
.PHONY: help
help: ## Show this help message
	@echo "OADP CLI Makefile"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Build with different platforms:"
	@echo "  make build PLATFORM=linux/amd64"
	@echo "  make build PLATFORM=linux/arm64"
	@echo "  make build PLATFORM=darwin/amd64"
	@echo "  make build PLATFORM=darwin/arm64"
	@echo "  make build PLATFORM=windows/amd64"
	@echo ""
	@echo "Release commands:"
	@echo "  make release-build         # Build binaries for all platforms"
	@echo "  make release-archives      # Create tar.gz archives for all platforms"

# Build targets
.PHONY: build
build: ## Build the kubectl plugin binary (use PLATFORM=os/arch for cross-compilation)
	@if [ -n "$(PLATFORM)" ]; then \
		echo "Building $(BINARY_NAME) for $(PLATFORM)..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME)-$(GOOS)-$(GOARCH) .; \
		echo "✅ Built $(BINARY_NAME)-$(GOOS)-$(GOARCH) successfully!"; \
	else \
		echo "Building $(BINARY_NAME) for current platform ($$(go env GOOS)/$$(go env GOARCH))..."; \
		go build -o $(BINARY_NAME) .; \
		echo "✅ Built $(BINARY_NAME) successfully!"; \
	fi

# Installation targets
.PHONY: install
install: build ## Build and install the kubectl plugin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	mv $(BINARY_NAME) $(INSTALL_PATH)/
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
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-* $(BINARY_NAME)-darwin-* $(BINARY_NAME)-windows-*
	@rm -f *.tar.gz *.sha256
	@rm -f oadp-*.yaml
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
	@if kubectl plugin list 2>/dev/null | grep -q "kubectl-oadp"; then \
		echo "  ✅ kubectl-oadp plugin is installed and accessible"; \
		echo "  Version check:"; \
		kubectl oadp version 2>/dev/null || echo "    (version command not available)"; \
	else \
		echo "  ❌ kubectl-oadp plugin is NOT accessible"; \
		echo "  Available plugins:"; \
		kubectl plugin list 2>/dev/null | head -5 || echo "    (no plugins found or kubectl not available)"; \
	fi

# Release targets
.PHONY: release-build
release-build: ## Build binaries for all platforms
	@echo "Building release binaries..."
	@platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64"); \
	for platform in $${platforms[@]}; do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name="$(BINARY_NAME).exe"; \
		else \
			binary_name="$(BINARY_NAME)"; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -o $$binary_name .; \
		echo "✅ Built $$binary_name for $$GOOS/$$GOARCH"; \
		mv $$binary_name $(BINARY_NAME)-$$GOOS-$$GOARCH$${binary_name#$(BINARY_NAME)}; \
	done
	@echo "✅ All release binaries built successfully!"

.PHONY: release-archives
release-archives: release-build ## Create tar.gz archives for all platforms (includes LICENSE)
	@echo "Creating release archives..."
	@if [ ! -f LICENSE ]; then \
		echo "❌ LICENSE file not found! Please ensure LICENSE file exists."; \
		exit 1; \
	fi
	@platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64"); \
	for platform in $${platforms[@]}; do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name="$(BINARY_NAME).exe"; \
		else \
			binary_name="$(BINARY_NAME)"; \
		fi; \
		archive_name="$(BINARY_NAME)-$$GOOS-$$GOARCH.tar.gz"; \
		echo "Creating $$archive_name..."; \
		tar czf $$archive_name LICENSE $(BINARY_NAME)-$$GOOS-$$GOARCH$${binary_name#$(BINARY_NAME)}; \
		sha256sum $$archive_name > $$archive_name.sha256; \
		echo "✅ Created $$archive_name with LICENSE"; \
	done
	@echo "✅ All release archives created successfully!"
	@echo "📦 Archives created:"
	@ls -la *.tar.gz
	@echo "🔐 SHA256 checksums:"
	@ls -la *.sha256

.PHONY: release
release: release-archives ## Build and create release archives for all platforms
	@echo "🚀 Release build complete! Archives ready for distribution."

.PHONY: krew-manifest
krew-manifest: release-archives ## Generate Krew plugin manifest with SHA256 checksums
	@echo "Generating Krew plugin manifest with SHA256 checksums..."
	@if [ ! -f oadp.yaml ]; then \
		echo "❌ oadp.yaml manifest template not found!"; \
		exit 1; \
	fi
	@cp oadp.yaml oadp-$(VERSION).yaml
	@echo "Updating version and URLs..."
	@sed -i '' "s/version: v1.0.0/version: $(VERSION)/" oadp-$(VERSION).yaml
	@sed -i '' "s|download/v1.0.0/|download/$(VERSION)/|g" oadp-$(VERSION).yaml
	@echo "Updating SHA256 checksums..."
	@if [ -f kubectl-oadp-linux-amd64.tar.gz.sha256 ]; then \
		sha256=$$(cat kubectl-oadp-linux-amd64.tar.gz.sha256 | cut -d' ' -f1); \
		sed -i '' "/os: linux/,/bin: kubectl-oadp/{/arch: amd64/,/bin: kubectl-oadp/{s/sha256: \"\"/sha256: \"$$sha256\"/;}}" oadp-$(VERSION).yaml; \
		echo "  ✅ linux/amd64: $$sha256"; \
	fi
	@if [ -f kubectl-oadp-linux-arm64.tar.gz.sha256 ]; then \
		sha256=$$(cat kubectl-oadp-linux-arm64.tar.gz.sha256 | cut -d' ' -f1); \
		sed -i '' "/os: linux/,/bin: kubectl-oadp/{/arch: arm64/,/bin: kubectl-oadp/{s/sha256: \"\"/sha256: \"$$sha256\"/;}}" oadp-$(VERSION).yaml; \
		echo "  ✅ linux/arm64: $$sha256"; \
	fi
	@if [ -f kubectl-oadp-darwin-amd64.tar.gz.sha256 ]; then \
		sha256=$$(cat kubectl-oadp-darwin-amd64.tar.gz.sha256 | cut -d' ' -f1); \
		sed -i '' "/os: darwin/,/bin: kubectl-oadp/{/arch: amd64/,/bin: kubectl-oadp/{s/sha256: \"\"/sha256: \"$$sha256\"/;}}" oadp-$(VERSION).yaml; \
		echo "  ✅ darwin/amd64: $$sha256"; \
	fi
	@if [ -f kubectl-oadp-darwin-arm64.tar.gz.sha256 ]; then \
		sha256=$$(cat kubectl-oadp-darwin-arm64.tar.gz.sha256 | cut -d' ' -f1); \
		sed -i '' "/os: darwin/,/bin: kubectl-oadp/{/arch: arm64/,/bin: kubectl-oadp/{s/sha256: \"\"/sha256: \"$$sha256\"/;}}" oadp-$(VERSION).yaml; \
		echo "  ✅ darwin/arm64: $$sha256"; \
	fi
	@if [ -f kubectl-oadp-windows-amd64.tar.gz.sha256 ]; then \
		sha256=$$(cat kubectl-oadp-windows-amd64.tar.gz.sha256 | cut -d' ' -f1); \
		sed -i '' "/os: windows/,/bin: kubectl-oadp.exe/{/arch: amd64/,/bin: kubectl-oadp.exe/{s/sha256: \"\"/sha256: \"$$sha256\"/;}}" oadp-$(VERSION).yaml; \
		echo "  ✅ windows/amd64: $$sha256"; \
	fi
	@echo "✅ Krew manifest generated: oadp-$(VERSION).yaml"
	@echo "📝 Review the manifest and update the GitHub release URLs as needed."

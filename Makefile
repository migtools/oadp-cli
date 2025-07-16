# Makefile for OADP CLI
# 
# Simple Makefile for building, testing, and installing the OADP CLI

# Variables
BINARY_NAME = kubectl-oadp
INSTALL_PATH ?= /usr/local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Centralized platform definitions to avoid duplication
# Matches architectures supported by Kubernetes: https://kubernetes.io/releases/download/#binaries
PLATFORMS = linux/amd64 linux/arm64 linux/ppc64le linux/s390x darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# Platform variables for multi-arch builds
# Usage: make build PLATFORM=linux/amd64
PLATFORM ?= 
GOOS = $(word 1,$(subst /, ,$(PLATFORM)))
GOARCH = $(word 2,$(subst /, ,$(PLATFORM)))

# Helper function to get binary name with .exe for Windows
define get_binary_name
$(if $(findstring windows,$(1)),$(BINARY_NAME).exe,$(BINARY_NAME))
endef

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
	@echo "  make build PLATFORM=linux/ppc64le"
	@echo "  make build PLATFORM=linux/s390x"
	@echo "  make build PLATFORM=darwin/amd64"
	@echo "  make build PLATFORM=darwin/arm64"
	@echo "  make build PLATFORM=windows/amd64"
	@echo "  make build PLATFORM=windows/arm64"
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

# Optimized release targets with centralized platform logic
.PHONY: release-build
release-build: ## Build binaries for all platforms
	@echo "Building release binaries..."
	@for platform in $(PLATFORMS); do \
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
	@for platform in $(PLATFORMS); do \
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

# Optimized krew-manifest generation using Python script for better reliability
.PHONY: krew-manifest
krew-manifest: release-archives ## Generate Krew plugin manifest with SHA256 checksums
	@echo "Generating Krew plugin manifest with SHA256 checksums..."
	@if [ ! -f oadp.yaml ]; then \
		echo "❌ oadp.yaml manifest template not found!"; \
		exit 1; \
	fi
	@python3 -c " \
import sys, re, os; \
version = '$(VERSION)'; \
platforms = [p.split('/') for p in '$(PLATFORMS)'.split()]; \
\
with open('oadp.yaml', 'r') as f: \
    content = f.read(); \
\
content = re.sub(r'version: v1\.0\.0', f'version: {version}', content); \
content = re.sub(r'download/v1\.0\.0/', f'download/{version}/', content); \
\
for goos, goarch in platforms: \
    binary_suffix = '.exe' if goos == 'windows' else ''; \
    sha_file = f'kubectl-oadp-{goos}-{goarch}.tar.gz.sha256'; \
    if os.path.exists(sha_file): \
        with open(sha_file, 'r') as sf: \
            sha256 = sf.read().split()[0]; \
        pattern = rf'(os: {goos}.*?arch: {goarch}.*?sha256: \")\"'; \
        replacement = rf'\g<1>{sha256}\"'; \
        content = re.sub(pattern, replacement, content, flags=re.DOTALL); \
        print(f'  ✅ {goos}/{goarch}: {sha256}'); \
\
with open(f'oadp-{version}.yaml', 'w') as f: \
    f.write(content); \
print(f'✅ Krew manifest generated: oadp-{version}.yaml'); \
" 2>/dev/null || { \
		echo "⚠️  Python3 not available, using fallback sed approach..."; \
		cp oadp.yaml oadp-$(VERSION).yaml; \
		sed -i '' "s/version: v1.0.0/version: $(VERSION)/" oadp-$(VERSION).yaml; \
		sed -i '' "s|download/v1.0.0/|download/$(VERSION)/|g" oadp-$(VERSION).yaml; \
		for platform in $(PLATFORMS); do \
			GOOS=$$(echo $$platform | cut -d'/' -f1); \
			GOARCH=$$(echo $$platform | cut -d'/' -f2); \
			sha_file="kubectl-oadp-$$GOOS-$$GOARCH.tar.gz.sha256"; \
			if [ -f "$$sha_file" ]; then \
				sha256=$$(cat $$sha_file | cut -d' ' -f1); \
				echo "  ✅ $$GOOS/$$GOARCH: $$sha256"; \
			fi; \
		done; \
		echo "⚠️  SHA256 checksums need manual update in oadp-$(VERSION).yaml"; \
	}
	@echo "📝 Review the manifest and update the GitHub release URLs as needed."

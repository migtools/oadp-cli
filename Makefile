# Makefile for OADP CLI
# 
# Simple Makefile for building, testing, and installing the OADP CLI

# Variables
BINARY_NAME = kubectl-oadp
INSTALL_PATH ?= $(HOME)/.local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VELERO_NAMESPACE ?= openshift-adp
ASSUME_DEFAULT ?= false

# Build information for version command
GIT_SHA := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_TREE_STATE := $(shell if [ -z "`git status --porcelain 2>/dev/null`" ]; then echo "clean"; else echo "dirty"; fi)
LDFLAGS := -X github.com/vmware-tanzu/velero/pkg/buildinfo.Version=$(VERSION) \
           -X github.com/vmware-tanzu/velero/pkg/buildinfo.GitSHA=$(GIT_SHA) \
           -X github.com/vmware-tanzu/velero/pkg/buildinfo.GitTreeState=$(GIT_TREE_STATE)

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
	@echo "Installation options:"
	@echo "  \033[36mmake install\033[0m                            # Install with auto-detection & interactive prompt"
	@echo "  \033[36mmake install ASSUME_DEFAULT=true\033[0m      # Install with default namespace (no detection/prompt)"
	@echo "  \033[36mmake install VELERO_NAMESPACE=velero\033[0m  # Install with custom namespace (no detection/prompt)"
	@echo "  \033[36mmake install-user\033[0m                       # Same as install (legacy alias)"
	@echo "  \033[36mmake install-bin\033[0m                        # Install to ~/bin (alternative, no sudo)"
	@echo "  \033[36mmake install-system\033[0m                     # Install to /usr/local/bin (requires sudo)"
	@echo ""
	@echo "Uninstall options:"
	@echo "  \033[36mmake uninstall\033[0m        # Remove from user locations (no sudo)"
	@echo "  \033[36mmake uninstall-system\033[0m # Remove from system locations (requires sudo)"
	@echo "  \033[36mmake uninstall-all\033[0m    # Remove from all locations (user + system)"
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
	@echo "Testing commands:"
	@echo "  make test              # Run all tests (unit + integration)"
	@echo "  make test-unit         # Run unit tests only"
	@echo "  make test-integration  # Run integration tests only"
	@echo ""
	@echo "Release commands:"
	@echo "  make release-build         # Build binaries for all platforms"
	@echo "  make release-archives      # Create tar.gz archives for all platforms"

# Build targets
.PHONY: build
build: ## Build the kubectl plugin binary (use PLATFORM=os/arch for cross-compilation)
	@if [ -n "$(PLATFORM)" ]; then \
		if [ "$(GOOS)" = "windows" ]; then \
			binary_suffix=".exe"; \
		else \
			binary_suffix=""; \
		fi; \
		echo "Building $(BINARY_NAME) for $(PLATFORM)..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-$(GOOS)-$(GOARCH)$$binary_suffix .; \
		echo "✅ Built $(BINARY_NAME)-$(GOOS)-$(GOARCH)$$binary_suffix successfully!"; \
	else \
		GOOS=$$(go env GOOS); \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name="$(BINARY_NAME).exe"; \
		else \
			binary_name="$(BINARY_NAME)"; \
		fi; \
		echo "Building $$binary_name for current platform ($$GOOS/$$(go env GOARCH))..."; \
		go build -ldflags "$(LDFLAGS)" -o $$binary_name .; \
		echo "✅ Built $$binary_name successfully!"; \
	fi

# Installation targets
.PHONY: install
install: build ## Build and install the kubectl plugin to ~/.local/bin (no sudo required)
	@GOOS=$$(go env GOOS); \
	if [ "$$GOOS" = "windows" ]; then \
		binary_name="$(BINARY_NAME).exe"; \
	else \
		binary_name="$(BINARY_NAME)"; \
	fi; \
	echo "Installing $$binary_name to $(INSTALL_PATH)..."; \
	mkdir -p $(INSTALL_PATH); \
	cp $$binary_name $(INSTALL_PATH)/
	@echo "✅ Installed to $(INSTALL_PATH)"
	@echo ""
	@echo "🔍 Checking PATH configuration..."
	@PATH_NEEDS_UPDATE=false; \
	PATH_UPDATED=false; \
	PATH_IN_CONFIG=false; \
	CURRENT_SESSION_NEEDS_UPDATE=false; \
	\
	if [[ ":$$PATH:" != *":$(INSTALL_PATH):"* ]]; then \
		PATH_NEEDS_UPDATE=true; \
		CURRENT_SESSION_NEEDS_UPDATE=true; \
		echo "   ├─ ⚠️  $(INSTALL_PATH) is not in your current PATH"; \
		\
		if [[ "$$SHELL" == */zsh* ]] && [[ -f "$$HOME/.zshrc" ]]; then \
			if ! grep -q '^[[:space:]]*export[[:space:]]*PATH.*\.local/bin' "$$HOME/.zshrc" 2>/dev/null; then \
				echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> "$$HOME/.zshrc"; \
				echo "   ├─ ✅ Added PATH export to ~/.zshrc"; \
				PATH_UPDATED=true; \
			else \
				echo "   ├─ ℹ️  PATH export already exists in ~/.zshrc"; \
				PATH_IN_CONFIG=true; \
			fi; \
		elif [[ "$$SHELL" == */bash* ]] && [[ -f "$$HOME/.bashrc" ]]; then \
			if ! grep -q '^[[:space:]]*export[[:space:]]*PATH.*\.local/bin' "$$HOME/.bashrc" 2>/dev/null; then \
				echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> "$$HOME/.bashrc"; \
				echo "   ├─ ✅ Added PATH export to ~/.bashrc"; \
				PATH_UPDATED=true; \
			else \
				echo "   ├─ ℹ️  PATH export already exists in ~/.bashrc"; \
				PATH_IN_CONFIG=true; \
			fi; \
		else \
			echo "   ├─ ⚠️  Unsupported shell or config file not found"; \
			echo "   │  └─ Manually add to your shell config: export PATH=\"$(INSTALL_PATH):$$PATH\""; \
			PATH_UPDATED=true; \
		fi; \
	else \
		echo "   └─ ✅ $(INSTALL_PATH) is already in PATH"; \
	fi; \
	\
	echo ""; \
	if [[ "$$CURRENT_SESSION_NEEDS_UPDATE" == "true" ]]; then \
		echo "🔧 To use kubectl oadp in this terminal session:"; \
		echo "   └─ export PATH=\"$(INSTALL_PATH):$$PATH\""; \
		echo ""; \
		echo "🔄 For future sessions:"; \
		if [[ "$$PATH_UPDATED" == "true" ]]; then \
			echo "   └─ Restart your terminal or run: source ~/.zshrc"; \
		elif [[ "$$PATH_IN_CONFIG" == "true" ]]; then \
			echo "   ├─ Restart your terminal or run: source ~/.zshrc"; \
			echo "   └─ (PATH export exists but may need shell restart)"; \
		else \
			echo "   └─ Add the PATH export to your shell configuration file"; \
		fi; \
	fi; \
	echo ""; \
		echo "📋 Configuration:"; \
	NAMESPACE=$(VELERO_NAMESPACE); \
	DETECTED=false; \
	if [[ "$(ASSUME_DEFAULT)" != "true" && "$(VELERO_NAMESPACE)" == "openshift-adp" ]]; then \
		echo ""; \
		echo "   🔍 Detecting OADP deployment in cluster..."; \
		DETECTED_NS=$$(kubectl get deployments --all-namespaces -o jsonpath='{.items[?(@.metadata.name=="openshift-adp-controller-manager")].metadata.namespace}' 2>/dev/null | head -1); \
		if [[ -n "$$DETECTED_NS" ]]; then \
			echo "   ├─ ✅ Found OADP controller in namespace: $$DETECTED_NS"; \
			NAMESPACE=$$DETECTED_NS; \
			DETECTED=true; \
		else \
			echo "   ├─ ⚠️  Could not find openshift-adp-controller-manager deployment"; \
		fi; \
		echo ""; \
		echo "   🔍 Looking for DataProtectionApplication (DPA) resources..."; \
		DETECTED_NS=$$(kubectl get dataprotectionapplication --all-namespaces -o jsonpath='{.items[0].metadata.namespace}' 2>/dev/null | head -1); \
		if [[ -n "$$DETECTED_NS" ]]; then \
			echo "   ├─ ✅ Found DPA resource in namespace: $$DETECTED_NS"; \
			NAMESPACE=$$DETECTED_NS; \
			DETECTED=true; \
		else \
			echo "   ├─ ⚠️  Could not find DataProtectionApplication resources"; \
		fi; \
		echo ""; \
		echo "   ⚠️  ⚠️  ⚠️"; \
		echo "   ├─ ❌ OADP Operator is not detected in the cluster"; \
		echo "   ├─ Fallback will check for Velero deployment as fallback"; \
		echo "   ├─ Consider using the velero cli instead"; \
		echo "   ⚠️  ⚠️  ⚠️"; \
		echo ""; \
		echo "   🔍 Looking for Velero deployment as fallback..."; \
		DETECTED_NS=$$(kubectl get deployments --all-namespaces -o jsonpath='{.items[?(@.metadata.name=="velero")].metadata.namespace}' 2>/dev/null | head -1); \
		if [[ -n "$$DETECTED_NS" ]]; then \
			echo "   ├─ ✅ Found Velero deployment in namespace: $$DETECTED_NS"; \
			NAMESPACE=$$DETECTED_NS; \
			DETECTED=true; \
		else \
			echo "   └─ ⚠️  Could not detect OADP or Velero deployment in cluster"; \
		fi; \
		if [[ "$$DETECTED" == "false" ]]; then \
			echo "   🤔 Which namespace should admin commands use for Velero resources?"; \
			echo "   │  └─ (Common options: openshift-adp, velero, oadp)"; \
			echo ""; \
			printf "   Enter namespace [default: $(VELERO_NAMESPACE)]: "; \
			read -r user_input; \
			if [[ -n "$$user_input" ]]; then \
				NAMESPACE=$$user_input; \
			fi; \
		fi; \
		echo ""; \
	fi; \
		echo "   ├─ Setting Velero namespace to: $$NAMESPACE"; \
		GOOS=$$(go env GOOS); \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name="$(BINARY_NAME).exe"; \
		else \
			binary_name="$(BINARY_NAME)"; \
		fi; \
		$(INSTALL_PATH)/$$binary_name client config set namespace=$$NAMESPACE 2>/dev/null || true; \
		echo "   └─ ✅ Client config initialized"; \
	echo ""; \
	echo "🧪 Verifying installation..."; \
	if [[ "$$CURRENT_SESSION_NEEDS_UPDATE" == "true" ]]; then \
		echo "   ├─ Temporarily updating PATH for verification"; \
		if PATH="$(INSTALL_PATH):$$PATH" command -v kubectl >/dev/null 2>&1; then \
			if PATH="$(INSTALL_PATH):$$PATH" kubectl plugin list 2>/dev/null | grep -q "kubectl-oadp"; then \
				echo "   ├─ ✅ Installation verified: kubectl oadp plugin is accessible"; \
				PATH="$(INSTALL_PATH):$$PATH" kubectl oadp version 2>/dev/null || echo "   │  └─ (Note: version command requires cluster access)"; \
			else \
				echo "   ├─ ❌ Installation verification failed: kubectl oadp plugin not found"; \
				echo "   │  └─ Try running: export PATH=\"$(INSTALL_PATH):$$PATH\""; \
			fi; \
		else \
			echo "   ├─ ⚠️  kubectl not found - cannot verify plugin accessibility"; \
			echo "   └─ Plugin installed to: $(INSTALL_PATH)/$$binary_name"; \
		fi; \
	else \
		if command -v kubectl >/dev/null 2>&1; then \
			if kubectl plugin list 2>/dev/null | grep -q "kubectl-oadp"; then \
				echo "   ├─ ✅ Installation verified: kubectl oadp plugin is accessible"; \
				echo "   ├─ Running version command..."; \
				echo ""; \
				kubectl oadp version 2>/dev/null || echo "   │  └─ (Note: version command requires cluster access)"; \
			else \
				echo "   └─ ❌ Installation verification failed: kubectl oadp plugin not found"; \
			fi; \
		else \
			echo "   ├─ ⚠️  kubectl not found - cannot verify plugin accessibility"; \
			echo "   └─ Plugin installed to: $(INSTALL_PATH)/$$binary_name"; \
		fi; \
	fi; \

.PHONY: install-user
install-user: build ## Build and install the kubectl plugin to ~/.local/bin (no sudo required)
	@echo "Installing $(BINARY_NAME) to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/
	@echo "✅ Installed to ~/.local/bin"
	@echo "Add to PATH: export PATH=\"\$$HOME/.local/bin:\$$PATH\""
	@echo "Test: kubectl oadp --help"

.PHONY: install-bin
install-bin: build ## Build and install the kubectl plugin to ~/bin (no sudo required)
	@echo "Installing $(BINARY_NAME) to ~/bin..."
	@mkdir -p ~/bin
	cp $(BINARY_NAME) ~/bin/
	@echo "✅ Installed to ~/bin"
	@echo "Add to PATH: export PATH=\"\$$HOME/bin:\$$PATH\""
	@echo "Test: kubectl oadp --help"

.PHONY: install-system
install-system: build ## Build and install the kubectl plugin to /usr/local/bin (requires sudo)
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo mv $(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin"
	@echo "Test: kubectl oadp --help"

.PHONY: uninstall
uninstall: ## Uninstall the kubectl plugin from user locations
	@echo "Removing $(BINARY_NAME) from user locations..."
	@removed=false; \
	if [ -f "$(INSTALL_PATH)/$(BINARY_NAME)" ]; then \
		rm -f "$(INSTALL_PATH)/$(BINARY_NAME)"; \
		echo "✅ Removed from $(INSTALL_PATH)"; \
		removed=true; \
	fi; \
	if [ -f "$$HOME/.local/bin/$(BINARY_NAME)" ] && [ "$(INSTALL_PATH)" != "$$HOME/.local/bin" ]; then \
		rm -f "$$HOME/.local/bin/$(BINARY_NAME)"; \
		echo "✅ Removed from ~/.local/bin"; \
		removed=true; \
	fi; \
	if [ -f "$$HOME/bin/$(BINARY_NAME)" ] && [ "$(INSTALL_PATH)" != "$$HOME/bin" ]; then \
		rm -f "$$HOME/bin/$(BINARY_NAME)"; \
		echo "✅ Removed from ~/bin"; \
		removed=true; \
	fi; \
	if [ "$$removed" = "false" ]; then \
		echo "⚠️  Not found in user locations"; \
	fi

.PHONY: uninstall-system
uninstall-system: ## Uninstall the kubectl plugin from system locations (requires sudo)
	@echo "Removing $(BINARY_NAME) from system locations..."
	@removed=false; \
	if [ -f "/usr/local/bin/$(BINARY_NAME)" ]; then \
		sudo rm -f "/usr/local/bin/$(BINARY_NAME)"; \
		echo "✅ Removed from /usr/local/bin"; \
		removed=true; \
	fi; \
	if [ -f "/usr/bin/$(BINARY_NAME)" ]; then \
		sudo rm -f "/usr/bin/$(BINARY_NAME)"; \
		echo "✅ Removed from /usr/bin"; \
		removed=true; \
	fi; \
	if [ "$$removed" = "false" ]; then \
		echo "⚠️  Not found in system locations"; \
	fi

.PHONY: uninstall-all
uninstall-all: ## Uninstall the kubectl plugin from all locations (user + system)
	@make --no-print-directory uninstall
	@make --no-print-directory uninstall-system

# Testing targets
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@echo "🧪 Running unit tests..."
	go test ./cmd/... ./internal/...
	@echo "🔗 Running integration tests..."
	go test . -v
	@echo "✅ Tests completed!"

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test ./cmd/... ./internal/...
	@echo "✅ Unit tests completed!"

.PHONY: test-integration
test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	go test . -v
	@echo "✅ Integration tests completed!"

# Cleanup targets
.PHONY: clean
clean: ## Remove built binaries
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-linux-* $(BINARY_NAME)-darwin-* $(BINARY_NAME)-windows-*
	@rm -f *.tar.gz *.sha256
	@rm -f oadp-*.yaml oadp-*.yaml.tmp
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
	@echo "Building release binaries for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		if [ -n "$(VERSION)" ]; then \
			version_suffix="_$(VERSION)"; \
		else \
			version_suffix=""; \
		fi; \
		if [ "$$GOOS" = "windows" ]; then \
			output_name="$(BINARY_NAME)$${version_suffix}_$${GOOS}_$${GOARCH}.exe"; \
		else \
			output_name="$(BINARY_NAME)$${version_suffix}_$${GOOS}_$${GOARCH}"; \
		fi; \
		echo "Building $$output_name..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "$(LDFLAGS)" -o $$output_name .; \
		echo "✅ Built $$output_name"; \
	done
	@echo "✅ All release binaries created successfully!"

.PHONY: release-archives
release-archives: release-build ## Create tar.gz archives with SHA256 checksums for all platforms
	@echo "Creating tar.gz archives with simple binary names..."
	@if [ ! -f LICENSE ]; then \
		echo "❌ LICENSE file not found! Please ensure LICENSE file exists."; \
		exit 1; \
	fi
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		if [ -n "$(VERSION)" ]; then \
			version_suffix="_$(VERSION)"; \
		else \
			version_suffix=""; \
		fi; \
		if [ "$$GOOS" = "windows" ]; then \
			platform_binary="$(BINARY_NAME)$${version_suffix}_$${GOOS}_$${GOARCH}.exe"; \
			simple_binary="$(BINARY_NAME).exe"; \
		else \
			platform_binary="$(BINARY_NAME)$${version_suffix}_$${GOOS}_$${GOARCH}"; \
			simple_binary="$(BINARY_NAME)"; \
		fi; \
		archive_name="$(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.tar.gz"; \
		echo "Creating $$archive_name..."; \
		cp $$platform_binary $$simple_binary; \
		tar czf $$archive_name LICENSE $$simple_binary; \
		rm $$simple_binary; \
		echo "✅ Created $$archive_name"; \
	done
	@echo ""
	@echo "Generating SHA256 checksums..."
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		archive_name="$(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.tar.gz"; \
		sha256sum $$archive_name > $$archive_name.sha256; \
		echo "✅ Generated checksum for $$archive_name"; \
	done
	@echo "✅ All SHA256 checksums generated!"
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
    sha_file = f'kubectl-oadp_${VERSION}_{goos}_{goarch}.tar.gz.sha256'; \
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
	# Use portable sed approach (works on both BSD/macOS and GNU/Linux) \
	sed "s/version: v1.0.0/version: $(VERSION)/" oadp-$(VERSION).yaml > oadp-$(VERSION).yaml.tmp && mv oadp-$(VERSION).yaml.tmp oadp-$(VERSION).yaml; \
	sed "s|download/v1.0.0/|download/$(VERSION)/|g" oadp-$(VERSION).yaml > oadp-$(VERSION).yaml.tmp && mv oadp-$(VERSION).yaml.tmp oadp-$(VERSION).yaml; \
		for platform in $(PLATFORMS); do \
			GOOS=$$(echo $$platform | cut -d'/' -f1); \
			GOARCH=$$(echo $$platform | cut -d'/' -f2); \
			sha_file="kubectl-oadp_${VERSION}_$$GOOS_$$GOARCH.tar.gz.sha256"; \
			if [ -f "$$sha_file" ]; then \
				sha256=$$(cat $$sha_file | cut -d' ' -f1); \
				echo "  ✅ $$GOOS/$$GOARCH: $$sha256"; \
			fi; \
		done; \
		echo "⚠️  SHA256 checksums need manual update in oadp-$(VERSION).yaml"; \
	}
	@echo "📝 Review the manifest and update the GitHub release URLs as needed."

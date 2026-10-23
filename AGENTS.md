# AGENTS.md — AI Agent Instructions for migtools/oadp-cli

## Project Overview
OADP CLI is a kubectl plugin (`kubectl oadp`) that provides a command-line interface for OADP (OpenShift API for Data Protection) operations. It simplifies backup, restore, and data protection management tasks. Supports cross-architecture builds and distributed as platform-specific binaries.

- **Primary Language**: Go
- **Module**: `github.com/migtools/oadp-cli`
- **Default Branch**: `oadp-dev`

## Build Instructions
```bash
# Build the kubectl plugin binary
make build

# Build and install to ~/.local/bin
make install

# Install to ~/bin
make install-bin

# Install to /usr/local/bin (requires sudo)
make install-system

# Build for a specific platform
make build PLATFORM=linux/amd64

# Build for all platforms (release)
make release-build

# Create release archives with checksums
make release-archives
```

## Test Instructions
```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run specific tests
go test ./internal/... -run TestName
go test ./cmd/... -run TestName

# Vet code
go vet ./...
```

## Linting
```bash
# Run golangci-lint
make lint

# Run golangci-lint with auto-fix and formatting
make lint-fix
```

Configuration: `.golangci.yml`

## Code Conventions
- kubectl plugin pattern (binary named `kubectl-oadp`)
- Commands in `cmd/` directory using cobra
- Internal packages in `internal/`
- Documentation in `docs/`
- Follow Go conventions for CLI tools
- Error messages should be user-friendly (this is a user-facing tool)

## Project Structure
```
cmd/           - CLI command definitions (cobra commands)
  root.go      - Root command and global flags
  backup/      - Backup subcommands
  restore/     - Restore subcommands
internal/      - Private packages
  client/      - Kubernetes client helpers
  util/        - Utility functions
docs/          - User documentation and guides
```

## CI/CD
- GitHub Actions workflows in `.github/workflows/`:
  - `cross-arch-build-test.yml` — Cross-architecture build testing
  - `lint.yml` — Linting checks
  - `test.yml` — Test suite
  - `release.yml` — Release automation
- Linter config: `.golangci.yml`
- Reproduce CI locally:
  ```bash
  make build
  make lint
  make test
  ```

## Common Tasks

### Adding a new CLI command
1. Create a new cobra command in `cmd/`
2. Add subcommand to the parent command registration
3. Implement logic in `internal/` packages
4. Add unit tests
5. Update documentation in `docs/`

### Uninstalling
```bash
make uninstall       # User locations
make uninstall-system # System locations
make uninstall-all   # All locations
```

### Creating a release
```bash
make release  # Builds all platforms and creates tar.gz archives with SHA256 checksums
```

### Checking build status
```bash
make status  # Shows build status and installation info
```

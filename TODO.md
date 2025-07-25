# DC-Update Go Conversion Plan

This document outlines the steps to convert the Node.js `dc-update` CLI tool to Go while maintaining all existing functionality and terminal output.

## Project Setup

- [x] Initialize Go module with `go mod init dc-update`
- [x] Create standard Go project structure:
  - [x] `cmd/dc-update/main.go` - CLI entry point
  - [x] `internal/` - Internal packages
  - [x] `pkg/` - Public packages (if needed)
- [x] Set up `go.mod` and `go.sum` files
- [x] Create `.gitignore` for Go project
- [x] Update `CLAUDE.md` with Go-specific instructions

## Dependency Research & Selection

- [x] Research Go Docker API clients (replace `dockerode`)
  - [x] Evaluate `github.com/docker/docker/client`
  - [x] Test basic Docker operations (list containers, inspect, etc.)
- [x] Research CLI parsing libraries (replace `meow`)
  - [x] Evaluate `github.com/spf13/cobra`
  - [x] Evaluate `github.com/urfave/cli/v2`
  - [x] Choose based on feature compatibility
- [x] Research terminal spinner libraries (replace `ora`)
  - [x] Evaluate `github.com/briandowns/spinner`
  - [x] Evaluate `github.com/schollz/progressbar/v3`
  - [x] Test spinner functionality and output formatting
- [x] Research docker-compose integration options
  - [x] Plan to execute `docker-compose` commands via `os/exec`
  - [x] Research YAML parsing for docker-compose.yml files

## Core Function Implementations

### CLI Interface
- [x] Implement CLI argument parsing with chosen library
- [x] Add flag definitions:
  - [x] `--file, -f` (string) - Path to docker-compose.yml file
  - [x] `--build, -b` (string slice) - Container names to build
  - [x] `--show-warnings` (bool) - Show warnings for non-running containers
- [x] Handle positional arguments for container names
- [x] Implement usage/help text matching original format
- [x] Add validation for docker-compose file existence

### Docker Compose Integration
- [x] Implement `getServiceNames()` function
  - [x] Execute `docker-compose config --services` command
  - [x] Parse output to extract service names
  - [x] Handle errors and empty results
- [x] Implement `getCurrentContainerId()` function
  - [x] Execute `docker-compose ps -q [service_name]` command
  - [x] Parse output to get container ID
  - [x] Handle non-running containers
- [x] Implement container operations:
  - [x] `stopContainer()` - Execute `docker-compose stop [service]`
  - [x] `removeContainer()` - Execute `docker-compose rm [service]`
  - [x] `startContainer()` - Execute `docker-compose up -d [service]`
  - [x] `pullContainer()` - Execute `docker-compose pull [service]`
  - [x] `buildContainers()` - Execute `docker-compose build --pull [services...]`

### Docker API Integration
- [x] Initialize Docker client connection
- [x] Implement `getCurrentImageId()` function
  - [x] Use Docker client to inspect container
  - [x] Extract current image ID from container info
  - [x] Handle SHA256 prefix parsing
- [x] Implement `getLatestImageId()` function
  - [x] Pull latest image using docker-compose
  - [x] Use Docker client to list images by reference
  - [x] Find most recently created image
  - [x] Extract and return image ID

### Core Business Logic
- [x] Implement `updateContainer()` function
  - [x] Initialize spinner with container name
  - [x] Get current container ID
  - [x] Handle non-running container warnings
  - [x] Compare current vs latest image IDs
  - [x] Restart container if image differs
  - [x] Update spinner text during operations
  - [x] Show success/failure messages
- [x] Implement `restartContainer()` function
  - [x] Stop container
  - [x] Remove container
  - [x] Start container
  - [x] Handle errors at each step

### Terminal Output & UX
- [x] Implement spinner functionality
  - [x] Start spinner with initial message
  - [x] Update spinner text during operations
  - [x] Success messages with checkmark
  - [x] Warning messages with warning symbol
  - [x] Error messages with X symbol
- [x] Implement `warnIfEnabled()` function
  - [x] Show warnings based on `--show-warnings` flag
  - [x] Proper spinner state management
- [x] Match original color scheme and formatting
- [x] Ensure proper terminal cleanup on exit

## Main Application Flow
- [x] Implement main execution logic
  - [x] Parse CLI arguments and flags
  - [x] Validate docker-compose file existence
  - [x] Determine service names (from args or all services)
  - [x] Handle build containers if specified
  - [x] Process each container sequentially
  - [x] Proper error handling and exit codes

## Error Handling & Edge Cases
- [x] Handle missing docker-compose file
- [x] Handle docker daemon connectivity issues
- [x] Handle non-existent containers/services
- [x] Handle docker-compose command failures
- [x] Handle Docker API errors
- [x] Implement proper exit codes for different error conditions
- [x] Add helpful error messages matching original behavior

## Testing & Validation
- [x] Create test docker-compose.yml for testing
- [x] Test CLI argument parsing
- [x] Test with running containers
- [x] Test with stopped containers
- [x] Test with non-existent containers
- [x] Test build functionality
- [x] Test file path handling (relative/absolute)
- [x] Test warning flag behavior
- [x] Compare output format with original Node.js version

## Build & Distribution
- [x] Install and configure GoReleaser
  - [x] Install GoReleaser CLI (`go install github.com/goreleaser/goreleaser@latest`)
  - [x] Initialize GoReleaser config with `goreleaser init`
  - [x] Configure `.goreleaser.yaml` for project needs
- [x] Set up cross-compilation and platform targets
  - [x] Configure supported OS/architecture combinations (linux, darwin, windows)
  - [x] Set up ARM64 and AMD64 builds
  - [x] Configure build flags and linker settings
- [x] Configure version management and Git integration
  - [x] Set up semantic versioning with Git tags
  - [x] Configure version embedding in binary (`-ldflags`)
  - [x] Set up Git tag validation and release triggers
- [x] Set up package manager integration
  - [x] Configure Homebrew tap for macOS installation
  - [x] Set up Scoop bucket for Windows installation
  - [ ] Configure AUR package for Arch Linux (optional)
- [x] Configure standalone binary distribution for direct downloads
  - [x] Configure GoReleaser to upload standalone executables alongside archives
  - [x] Set up individual binary uploads for each OS/arch combination
  - [x] Enable direct wget/curl download URLs (e.g., `wget https://github.com/user/dc-update/releases/latest/download/dc-update-linux-amd64`)
  - [x] Configure proper binary naming conventions for easy server deployment
  - [x] Verify standalone binaries work without extraction or additional setup
- [x] Configure release automation
  - [x] Set up GitHub Actions workflow for automated releases
  - [x] Configure release notes and changelog generation
  - [x] Set up draft releases and manual approval process
- [x] Set up container distribution (optional)
  - [x] Configure Docker image builds and registry pushes
  - [x] Set up multi-arch container images
- [x] Configure security and signing
  - [x] Set up binary signing for macOS and Windows
  - [x] Configure checksum generation and verification
  - [x] Set up SLSA provenance generation
- [x] Test release process
  - [x] Test local builds with `goreleaser build --snapshot`
  - [x] Test full release process with `goreleaser release --snapshot`
  - [x] Validate generated artifacts and packages

## Documentation Updates
- [x] Update README.md for Go installation and usage
- [x] Update CLAUDE.md with Go-specific development commands
- [x] Document new build and test commands
- [x] Update examples if needed

## Performance & Optimization
- [ ] Implement proper concurrency if beneficial
- [ ] Optimize Docker API calls
- [ ] Consider connection pooling/reuse
- [ ] Profile memory usage
- [ ] Optimize for startup time

## Final Verification
- [ ] Full functionality comparison with Node.js version
- [ ] Performance benchmarking
- [ ] Memory usage analysis
- [ ] Error handling verification
- [ ] Output format verification
- [ ] Cross-platform testing (if applicable)

## Migration Notes
- **Breaking Changes**: None expected - CLI interface should remain identical
- **Dependencies**: Go binary will be self-contained, no npm installation required
- **Performance**: Expected improvement in startup time and memory usage
- **Maintenance**: Simplified dependency management with Go modules
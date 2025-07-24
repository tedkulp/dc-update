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
- [ ] Initialize Docker client connection
- [ ] Implement `getCurrentImageId()` function
  - [ ] Use Docker client to inspect container
  - [ ] Extract current image ID from container info
  - [ ] Handle SHA256 prefix parsing
- [ ] Implement `getLatestImageId()` function
  - [ ] Pull latest image using docker-compose
  - [ ] Use Docker client to list images by reference
  - [ ] Find most recently created image
  - [ ] Extract and return image ID

### Core Business Logic
- [ ] Implement `updateContainer()` function
  - [ ] Initialize spinner with container name
  - [ ] Get current container ID
  - [ ] Handle non-running container warnings
  - [ ] Compare current vs latest image IDs
  - [ ] Restart container if image differs
  - [ ] Update spinner text during operations
  - [ ] Show success/failure messages
- [ ] Implement `restartContainer()` function
  - [ ] Stop container
  - [ ] Remove container
  - [ ] Start container
  - [ ] Handle errors at each step

### Terminal Output & UX
- [ ] Implement spinner functionality
  - [ ] Start spinner with initial message
  - [ ] Update spinner text during operations
  - [ ] Success messages with checkmark
  - [ ] Warning messages with warning symbol
  - [ ] Error messages with X symbol
- [ ] Implement `warnIfEnabled()` function
  - [ ] Show warnings based on `--show-warnings` flag
  - [ ] Proper spinner state management
- [ ] Match original color scheme and formatting
- [ ] Ensure proper terminal cleanup on exit

## Main Application Flow
- [ ] Implement main execution logic
  - [ ] Parse CLI arguments and flags
  - [ ] Validate docker-compose file existence
  - [ ] Determine service names (from args or all services)
  - [ ] Handle build containers if specified
  - [ ] Process each container sequentially
  - [ ] Proper error handling and exit codes

## Error Handling & Edge Cases
- [ ] Handle missing docker-compose file
- [ ] Handle docker daemon connectivity issues
- [ ] Handle non-existent containers/services
- [ ] Handle docker-compose command failures
- [ ] Handle Docker API errors
- [ ] Implement proper exit codes for different error conditions
- [ ] Add helpful error messages matching original behavior

## Testing & Validation
- [ ] Create test docker-compose.yml for testing
- [ ] Test CLI argument parsing
- [ ] Test with running containers
- [ ] Test with stopped containers
- [ ] Test with non-existent containers
- [ ] Test build functionality
- [ ] Test file path handling (relative/absolute)
- [ ] Test warning flag behavior
- [ ] Compare output format with original Node.js version

## Build & Distribution
- [ ] Create build scripts/Makefile
- [ ] Set up cross-compilation for multiple platforms
- [ ] Configure version information embedding
- [ ] Create installation instructions
- [ ] Update release process from npm to Go releases

## Documentation Updates
- [ ] Update README.md for Go installation and usage
- [ ] Update CLAUDE.md with Go-specific development commands
- [ ] Document new build and test commands
- [ ] Update examples if needed

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
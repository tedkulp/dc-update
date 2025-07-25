# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dc-update` is a CLI tool for updating large docker-compose based systems. It intelligently updates only containers that have newer images available, avoiding unnecessary restarts.

## Architecture

### Go Implementation (Current)
- **Main entry point**: `cmd/dc-update/main.go` - CLI application entry point
- **Project structure**: Standard Go layout with `cmd/`, `internal/`, and `pkg/` directories
- **Key packages**:
  - `internal/compose/` - Docker Compose integration via `os/exec`
  - `internal/docker/` - Docker API client wrapper
  - `internal/core/` - Core business logic and update orchestration
- **Dependencies**: 
  - `github.com/docker/docker/client` for Docker API access
  - `github.com/urfave/cli/v2` for CLI parsing and flags
  - `github.com/briandowns/spinner` for terminal spinners  
  - `os/exec` for docker-compose command execution
- **Core workflow**: 
  1. Get expected image name from docker-compose file for each service
  2. Get current running container's image ID
  3. Pull expected image and get its image ID
  4. Compare image IDs - if different, restart container
  5. Optional build step for containers that need building before update

### Node.js Implementation (Legacy)
- **Single-file application**: The main logic is in `index.js` with a simple executable wrapper in `bin/dc-update`
- **Dependencies**: Uses `docker-compose-tedkulp` (custom fork), `dockerode` for Docker API access, `meow` for CLI parsing, and `ora` for spinners

## Key Functions

### Compose Integration (`internal/compose/`)
- `GetServiceNames()`: Extracts service names from docker-compose file using `docker compose config --services`
- `GetCurrentContainerId()`: Gets container ID for a service using `docker compose ps -q [service]`
- `GetServiceImageName()`: Extracts expected image name from docker-compose config
- `PullContainer()`: Pulls image using `docker compose pull [service]`
- `BuildContainers()`: Builds containers using `docker compose build --pull [services...]`
- `StopContainer()`, `RemoveContainer()`, `StartContainer()`: Container lifecycle management

### Docker API Integration (`internal/docker/`)
- `GetCurrentImageId()`: Gets current container's image ID via Docker API
- `GetImageId()`: Gets image ID for a specific image reference (name:tag)
- `NewClient()`: Creates Docker API client with connection testing

### Core Logic (`internal/core/`)
- `UpdateContainer()`: Main update logic with image comparison and spinner UI
- `RestartContainer()`: Orchestrates stop/remove/start sequence

## CLI Usage

The tool accepts container names as arguments and supports:
- `--file, -f`: Path to docker-compose.yml file
- `--build, -b`: Container names to build (can be used multiple times)
- `--show-warnings`: Show warnings for non-running containers

## Development Commands

### Go Implementation (Current)

#### Basic Development
- `make build` or `go build -o dc-update cmd/dc-update/main.go`: Build binary for current platform
- `make dev ARGS="--help"` or `go run cmd/dc-update/main.go --help`: Run without building (for quick testing)
- `make install` or `go install ./cmd/dc-update`: Install globally for testing (`$GOPATH/bin/dc-update`)
- `make deps` or `go mod tidy`: Manage dependencies and clean up unused modules

#### Testing & Quality
- `make test` or `go test ./...`: Run all tests (when implemented)
- `make lint` or `go vet ./...`: Run Go's built-in static analyzer
- `make fmt` or `go fmt ./...`: Format all Go code according to standards

#### Cross-Platform Building
- `make release` or `goreleaser build --snapshot --clean`: Build for all platforms locally
- `goreleaser release --snapshot --clean`: Test full release process locally
- `make clean`: Clean build artifacts

#### Release Process
- Create and push Git tag: `git tag v1.0.0 && git push origin v1.0.0`
- GitHub Actions automatically runs GoReleaser to create release with:
  - Cross-platform binaries (Linux, macOS, Windows - AMD64 & ARM64)
  - Standalone binary downloads for direct wget/curl installation
  - Homebrew and Scoop package updates
  - Docker multi-arch images
  - Checksums and release notes

### Project Structure
```
dc-update/
├── cmd/dc-update/main.go          # CLI entry point
├── internal/
│   ├── compose/compose.go         # Docker Compose integration
│   ├── docker/docker.go           # Docker API client
│   └── core/core.go               # Business logic
├── .goreleaser.yaml               # Release configuration
├── .github/workflows/release.yml  # CI/CD pipeline
└── Dockerfile                     # Container image build
```

### Node.js Implementation (Legacy)
This project has no test suite, linting, or build commands configured in package.json. Development is straightforward:

- `npm install`: Install dependencies
- `node bin/dc-update --help`: Test the CLI locally
- `npm link`: Install globally for testing
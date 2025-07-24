# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dc-update` is a CLI tool for updating large docker-compose based systems. It intelligently updates only containers that have newer images available, avoiding unnecessary restarts.

## Architecture

### Go Implementation (Current)
- **Main entry point**: `cmd/dc-update/main.go` - CLI application entry point
- **Project structure**: Standard Go layout with `cmd/`, `internal/`, and `pkg/` directories
- **Dependencies**: 
  - `github.com/docker/docker/client` for Docker API access
  - `github.com/urfave/cli/v2` for CLI parsing and flags
  - `github.com/briandowns/spinner` for terminal spinners  
  - `os/exec` for docker-compose command execution
  - `gopkg.in/yaml.v3` for YAML parsing (if needed)
- **Core workflow**: 
  1. Parse docker-compose file to get service names
  2. For each container: pull latest image, compare image IDs, restart only if different
  3. Optional build step for containers that need building before update

### Node.js Implementation (Legacy)
- **Single-file application**: The main logic is in `index.js` with a simple executable wrapper in `bin/dc-update`
- **Dependencies**: Uses `docker-compose-tedkulp` (custom fork), `dockerode` for Docker API access, `meow` for CLI parsing, and `ora` for spinners

## Key Functions

- `getServiceNames()`: Extracts service names from docker-compose file
- `getCurrentContainerId()` / `getCurrentImageId()`: Gets current container/image info
- `getLatestImageId()`: Pulls and identifies latest available image
- `updateContainer()`: Main update logic with image comparison
- `buildContainers()`: Builds specified containers with `--pull` flag

## CLI Usage

The tool accepts container names as arguments and supports:
- `--file, -f`: Path to docker-compose.yml file
- `--build, -b`: Container names to build (can be used multiple times)
- `--show-warnings`: Show warnings for non-running containers

## Development Commands

### Go Implementation
- `go mod tidy`: Manage dependencies
- `go build -o dc-update cmd/dc-update/main.go`: Build binary
- `go run cmd/dc-update/main.go --help`: Test the CLI locally
- `go install ./cmd/dc-update`: Install globally for testing
- `go test ./...`: Run tests (when implemented)

### Node.js Implementation (Legacy)
This project has no test suite, linting, or build commands configured in package.json. Development is straightforward:

- `npm install`: Install dependencies
- `node bin/dc-update --help`: Test the CLI locally
- `npm link`: Install globally for testing

## Release

Uses `release-it` for automated releases to GitHub (configured in package.json).
# dc-update

An opinionated script for updating large docker-compose based systems

## Why?

Docker and docker-compose are great. Though, my home server currently has 40 running containers, which is kind of a nightmare to keep on top of. This program is the next step in the evolution of a [gross Bash script](https://gist.github.com/tedkulp/80c5c707827556ed74e04bb92d924531) that I used to keep it all up to date.

`dc-update` intelligently updates only containers that have newer images available or when the image reference changes (e.g., `node:16` → `node:18`), avoiding unnecessary restarts and downtime.

## Install

### Binary Releases (Recommended)

Download the latest binary for your platform from the [releases page](https://github.com/tedkulp/dc-update/releases):

```bash
# Linux (x86_64)
wget https://github.com/tedkulp/dc-update/releases/latest/download/dc-update-linux-amd64 -O dc-update
chmod +x dc-update
sudo mv dc-update /usr/local/bin/

# Linux (ARM64)
wget https://github.com/tedkulp/dc-update/releases/latest/download/dc-update-linux-arm64 -O dc-update
chmod +x dc-update
sudo mv dc-update /usr/local/bin/

# macOS (Intel)
wget https://github.com/tedkulp/dc-update/releases/latest/download/dc-update-darwin-amd64 -O dc-update
chmod +x dc-update
sudo mv dc-update /usr/local/bin/

# macOS (Apple Silicon)
wget https://github.com/tedkulp/dc-update/releases/latest/download/dc-update-darwin-arm64 -O dc-update
chmod +x dc-update
sudo mv dc-update /usr/local/bin/

# Windows
# Download dc-update-windows-amd64.exe from the releases page
```

### Package Managers

#### Homebrew (macOS/Linux)
```bash
brew tap tedkulp/tap
brew install dc-update
```

#### Scoop (Windows)
```bash
scoop bucket add tedkulp https://github.com/tedkulp/scoop-bucket
scoop install dc-update
```

### Docker
```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(pwd):/workspace -w /workspace ghcr.io/tedkulp/dc-update:latest
```

### Build from Source
```bash
git clone https://github.com/tedkulp/dc-update
cd dc-update
go build -o dc-update cmd/dc-update/main.go
```

## Requirements

- Docker
- Docker Compose (v2 recommended, but v1 with `docker-compose` command also works)

## Usage

To update all running containers in the docker-compose file in the current directory:

```bash
dc-update
```

Only want to update specific containers?

```bash
dc-update container1 container2 container3
```

Is the docker-compose file in another location?

```bash
dc-update -f /path/to/docker-compose.yml
```

Show warnings for containers that aren't running:

```bash
dc-update --show-warnings
```

Get help with all available options:

```bash
dc-update --help
```

## Building Containers

Some containers need to be built before they can be updated. The `--build` option will pull and build specified containers first, then update them if changes were detected:

```bash
dc-update --build app --build proxy
```

**Note**: The build list and update list are separate. If you want to build **AND** update only specific containers, they must be specified separately:

```bash
dc-update --build app --build proxy app proxy
```

## How It Works

`dc-update` intelligently determines which containers need updating by:

1. **Checking the expected image** from your docker-compose.yml file
2. **Comparing with the running container's image** 
3. **Detecting changes** in image tags (e.g., `node:16` → `node:18`) or image updates
4. **Pulling the latest image** to ensure comparison accuracy
5. **Restarting only containers** with different images

This approach minimizes unnecessary restarts and downtime.

## Examples

### Basic Usage
```bash
# Update all running containers
dc-update

# Update specific containers
dc-update web api database

# Use different compose file
dc-update -f docker-compose.production.yml
```

### Building and Updating
```bash
# Build and update all containers
dc-update --build web --build api

# Build specific containers, update all
dc-update --build web --build api

# Build and update only specific containers
dc-update --build web --build api web api
```

### Debugging
```bash
# Show warnings for stopped containers
dc-update --show-warnings

# Check version
dc-update --version
```

## Migrating from Node.js Version

If you're upgrading from the previous Node.js version of `dc-update`:

1. **Uninstall the old version**: `npm uninstall -g dc-update`
2. **Install the new Go version** using any of the methods above
3. **No configuration changes needed** - the CLI interface is identical

### Benefits of the Go Version

- **Faster startup time** - No Node.js runtime overhead
- **Self-contained binary** - No dependencies to install
- **Better error handling** - More detailed error messages
- **Cross-platform releases** - Binaries for Linux, macOS, and Windows
- **Smaller resource footprint** - Lower memory usage

## Development

### Building from Source

```bash
git clone https://github.com/tedkulp/dc-update
cd dc-update
go mod tidy
go build -o dc-update cmd/dc-update/main.go
```

### Development Commands

Using the provided Makefile:
```bash
# Build for current platform
make build

# Run without building (for development)
make dev ARGS="--help"

# Run tests
make test

# Lint and format code
make lint

# Install globally for testing
make install

# Clean build artifacts
make clean

# Build for all platforms (requires GoReleaser)
make release
```

Or use Go commands directly:
```bash
# Build for current platform
go build -o dc-update cmd/dc-update/main.go

# Run without building
go run cmd/dc-update/main.go --help

# Install globally for testing
go install ./cmd/dc-update

# Run tests (when implemented)
go test ./...

# Build for all platforms (requires GoReleaser)
goreleaser build --snapshot --clean
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see the [LICENSE](LICENSE) file for details.

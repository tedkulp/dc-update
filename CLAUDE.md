# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dc-update` is a CLI tool for updating large docker-compose based systems. It intelligently updates only containers that have newer images available, avoiding unnecessary restarts.

## Architecture

- **Single-file application**: The main logic is in `index.js` with a simple executable wrapper in `bin/dc-update`
- **Dependencies**: Uses `docker-compose-tedkulp` (custom fork), `dockerode` for Docker API access, `meow` for CLI parsing, and `ora` for spinners
- **Core workflow**: 
  1. Parse docker-compose file to get service names
  2. For each container: pull latest image, compare image IDs, restart only if different
  3. Optional build step for containers that need building before update

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

This project has no test suite, linting, or build commands configured in package.json. Development is straightforward:

- `npm install`: Install dependencies
- `node bin/dc-update --help`: Test the CLI locally
- `npm link`: Install globally for testing

## Release

Uses `release-it` for automated releases to GitHub (configured in package.json).
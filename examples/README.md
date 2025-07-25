# dc-update Examples

This directory contains example configurations and usage scenarios for `dc-update`.

## Example Docker Compose File

The `docker-compose.example.yml` file demonstrates various types of services that `dc-update` can manage:

- **nginx**: Simple image reference with latest tag
- **api**: Explicitly versioned image (node:18-alpine)
- **database**: Database service with persistent storage
- **redis**: Cache service with data persistence
- **worker**: Same image as API but different command

## Usage Examples

### Update All Running Containers
```bash
dc-update -f docker-compose.example.yml
```

### Update Specific Services
```bash
dc-update -f docker-compose.example.yml nginx api
```

### Build and Update (if you have build services)
```bash
dc-update -f docker-compose.example.yml --build app
```

### Show Warnings for Stopped Containers
```bash
dc-update -f docker-compose.example.yml --show-warnings
```

## Scenarios dc-update Handles

### 1. Tag Changes
If you update `node:18-alpine` to `node:20-alpine` in the compose file, `dc-update` will detect this change and restart the affected containers.

### 2. Image Updates
When a new version of `nginx:alpine` is published, `dc-update` will detect the different image ID and restart the container.

### 3. Mixed Updates
You can update some containers while leaving others unchanged. Only containers with actual changes will be restarted.

### 4. Build Dependencies
For containers that need building (using `build:` instead of `image:`), use the `--build` flag to ensure they're rebuilt before update comparison.

## Best Practices

1. **Use specific tags** when possible (e.g., `postgres:15-alpine` instead of `postgres:latest`)
2. **Test changes** in development before updating production services
3. **Use --show-warnings** to identify stopped containers that won't be updated
4. **Backup data** before major updates, especially for database containers
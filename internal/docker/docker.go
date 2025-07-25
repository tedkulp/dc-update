package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client wraps the Docker API client with caching for performance
type Client struct {
	cli        *client.Client
	ctx        context.Context
	imageCache map[string]*types.ImageSummary  // Cache for image lookups
	containerCache map[string]*types.ContainerJSON // Cache for container inspections
}

// NewClient creates a new Docker API client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon - is Docker running? %w", err)
	}

	// Test the connection by pinging the daemon
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("Docker daemon is not responding - check Docker daemon status: %w", err)
	}

	return &Client{
		cli:        cli,
		ctx:        ctx,
		imageCache: make(map[string]*types.ImageSummary),
		containerCache: make(map[string]*types.ContainerJSON),
	}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// populateImageCache loads all images into cache for faster lookups
func (c *Client) populateImageCache() error {
	// Skip if cache is already populated
	if len(c.imageCache) > 0 {
		return nil
	}
	
	images, err := c.cli.ImageList(c.ctx, types.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Docker images: %w", err)
	}
	
	// Pre-allocate cache with estimated capacity
	estimatedCapacity := len(images) * 2 // Rough estimate for repo tags
	c.imageCache = make(map[string]*types.ImageSummary, estimatedCapacity)
	
	// Populate cache with all image references
	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			if repoTag != "<none>:<none>" {
				imageCopy := image // Create copy to avoid pointer issues
				c.imageCache[repoTag] = &imageCopy
			}
		}
	}
	
	return nil
}

// getContainerInspection gets container info with caching
func (c *Client) getContainerInspection(containerID string) (*types.ContainerJSON, error) {
	// Check cache first
	if cached, exists := c.containerCache[containerID]; exists {
		return cached, nil
	}
	
	// Not in cache, fetch from API
	containerJSON, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil, fmt.Errorf("container %s does not exist or is not running", containerID)
		}
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	
	// Cache the result
	c.containerCache[containerID] = &containerJSON
	return &containerJSON, nil
}

// GetCurrentImageId inspects a container and returns its current image ID
func (c *Client) GetCurrentImageId(containerID string) (string, error) {
	containerJSON, err := c.getContainerInspection(containerID)
	if err != nil {
		return "", err
	}

	// Extract image ID and remove 'sha256:' prefix if present
	imageID := containerJSON.Image
	if strings.HasPrefix(imageID, "sha256:") {
		imageID = strings.TrimPrefix(imageID, "sha256:")
	}

	return imageID, nil
}

// GetLatestImageId gets the container's image name and finds the latest image with that reference
func (c *Client) GetLatestImageId(containerID string) (string, error) {
	// First inspect the container to get its image name
	containerJSON, err := c.getContainerInspection(containerID)
	if err != nil {
		return "", err
	}

	// Get the image name from container config
	imageName := containerJSON.Config.Image
	if imageName == "" {
		return "", nil
	}

	// Use the optimized GetImageId method
	return c.GetImageId(imageName)
}

// GetImageId gets the image ID for a specific image reference (name:tag)
func (c *Client) GetImageId(imageName string) (string, error) {
	if imageName == "" {
		return "", fmt.Errorf("image name cannot be empty")
	}

	// Ensure image cache is populated
	if err := c.populateImageCache(); err != nil {
		return "", err
	}

	// Look up image in cache
	if image, exists := c.imageCache[imageName]; exists {
		// Remove sha256: prefix if present
		imageID := image.ID
		if strings.HasPrefix(imageID, "sha256:") {
			imageID = imageID[7:]
		}
		return imageID, nil
	}

	// If we didn't find it locally, return empty string (image may need to be pulled)
	return "", nil
}

// RefreshImageCache clears and repopulates the image cache
// This should be called after docker-compose pull operations
func (c *Client) RefreshImageCache() error {
	// Clear existing cache
	c.imageCache = make(map[string]*types.ImageSummary)
	
	// Repopulate cache
	return c.populateImageCache()
}
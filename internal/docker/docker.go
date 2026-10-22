package docker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/moby/moby/api/types/container"
	imagetypes "github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

// dockerAPI is the subset of the Docker client used by Client, allowing test fakes.
type dockerAPI interface {
	ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error)
	ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	Close() error
}

// Client wraps the Docker API client with caching for performance
type Client struct {
	mu             sync.RWMutex
	cli            dockerAPI
	ctx            context.Context
	imageCache     map[string]*imagetypes.Summary
	containerCache map[string]*container.InspectResponse
}

// NewClient creates a new Docker API client
func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon - is Docker running? %w", err)
	}

	// Test the connection by pinging the daemon
	ctx := context.Background()
	_, err = cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("Docker daemon is not responding - check Docker daemon status: %w", err)
	}

	return &Client{
		cli:        cli,
		ctx:        ctx,
		imageCache: make(map[string]*imagetypes.Summary),
		containerCache: make(map[string]*container.InspectResponse),
	}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// populateImageCache loads all images into cache for faster lookups
func (c *Client) populateImageCache() error {
	c.mu.RLock()
	populated := len(c.imageCache) > 0
	c.mu.RUnlock()
	if populated {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check now that we hold the write lock
	if len(c.imageCache) > 0 {
		return nil
	}

	result, err := c.cli.ImageList(c.ctx, client.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Docker images: %w", err)
	}
	images := result.Items

	estimatedCapacity := len(images) * 2
	c.imageCache = make(map[string]*imagetypes.Summary, estimatedCapacity)

	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			if repoTag != "<none>:<none>" {
				imageCopy := image
				c.imageCache[repoTag] = &imageCopy
			}
		}
	}

	return nil
}

// getContainerInspection gets container info with caching
func (c *Client) getContainerInspection(containerID string) (*container.InspectResponse, error) {
	c.mu.RLock()
	cached, exists := c.containerCache[containerID]
	c.mu.RUnlock()
	if exists {
		return cached, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check now that we hold the write lock
	if cached, exists = c.containerCache[containerID]; exists {
		return cached, nil
	}

	result, err := c.cli.ContainerInspect(c.ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil, fmt.Errorf("container %s does not exist or is not running", containerID)
		}
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	inspection := result.Container
	c.containerCache[containerID] = &inspection
	return &inspection, nil
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

// normalizeImageName ensures image names have a tag, defaulting to :latest if no tag is present.
// Docker images are stored in RepoTags with a tag, so a bare name like "nginx" won't match
// the cached "nginx:latest" entry without this normalization.
func normalizeImageName(name string) string {
	if !strings.Contains(name, ":") {
		return name + ":latest"
	}
	return name
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

	// Normalize image name by appending :latest if no tag is present
	// Docker images are always stored with a tag in RepoTags, so a bare
	// name like "nginx" won't match the cached "nginx:latest" entry.
	normalizedName := normalizeImageName(imageName)

	c.mu.RLock()
	image, exists := c.imageCache[normalizedName]
	c.mu.RUnlock()
	if exists {
		imageID := image.ID
		if strings.HasPrefix(imageID, "sha256:") {
			imageID = imageID[7:]
		}
		return imageID, nil
	}

	return "", nil
}

// RefreshImageCache clears and repopulates the image cache.
// This should be called after docker-compose pull operations.
// Both caches are cleared atomically and repopulated under a single write lock
// to prevent concurrent callers from observing a partially-refreshed state.
func (c *Client) RefreshImageCache() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.containerCache = make(map[string]*container.InspectResponse)

	result, err := c.cli.ImageList(c.ctx, client.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Docker images: %w", err)
	}
	images := result.Items

	c.imageCache = make(map[string]*imagetypes.Summary, len(images)*2)
	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			if repoTag != "<none>:<none>" {
				imageCopy := image
				c.imageCache[repoTag] = &imageCopy
			}
		}
	}

	return nil
}

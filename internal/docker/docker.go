package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client wraps the Docker API client
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient creates a new Docker API client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// GetCurrentImageId inspects a container and returns its current image ID
func (c *Client) GetCurrentImageId(containerID string) (string, error) {
	containerJSON, err := c.cli.ContainerInspect(c.ctx, containerID)
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
	containerJSON, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return "", err
	}

	// Get the image name from container config
	imageName := containerJSON.Config.Image
	if imageName == "" {
		return "", nil
	}

	// List images with this reference
	images, err := c.cli.ImageList(c.ctx, types.ImageListOptions{})
	if err != nil {
		return "", err
	}

	// Find the most recently created image with matching reference
	var latestImage *types.ImageSummary
	latestCreated := int64(0)

	for _, image := range images {
		// Check if this image matches our reference
		for _, repoTag := range image.RepoTags {
			if repoTag == imageName {
				if image.Created > latestCreated {
					latestCreated = image.Created
					imageCopy := image // Create a copy to avoid pointer issues
					latestImage = &imageCopy
				}
				break
			}
		}
	}

	if latestImage == nil {
		return "", nil
	}

	// Extract image ID and remove 'sha256:' prefix if present
	imageID := latestImage.ID
	if strings.HasPrefix(imageID, "sha256:") {
		imageID = strings.TrimPrefix(imageID, "sha256:")
	}

	return imageID, nil
}
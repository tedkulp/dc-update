package core

import (
	"fmt"
	"time"

	"dc-update/internal/compose"
	"dc-update/internal/docker"

	"github.com/briandowns/spinner"
)

// UpdaterOptions holds configuration for the updater
type UpdaterOptions struct {
	ShowWarnings bool
	ComposeOpts  *compose.Options
	DockerClient *docker.Client
}

// NewUpdaterOptions creates new updater options
func NewUpdaterOptions(composeFile string, showWarnings bool) (*UpdaterOptions, error) {
	composeOpts := compose.NewOptions(composeFile)
	
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	return &UpdaterOptions{
		ShowWarnings: showWarnings,
		ComposeOpts:  composeOpts,
		DockerClient: dockerClient,
	}, nil
}

// Close closes the Docker client connection
func (opts *UpdaterOptions) Close() error {
	return opts.DockerClient.Close()
}

// warnIfEnabled shows warnings based on the ShowWarnings flag
func (opts *UpdaterOptions) warnIfEnabled(s *spinner.Spinner, message string) {
	if opts.ShowWarnings {
		s.FinalMSG = fmt.Sprintf("⚠️  %s\n", message)
		s.Stop()
	} else {
		s.Stop()
	}
}

// RestartContainer stops, removes, and starts a container
func (opts *UpdaterOptions) RestartContainer(serviceName string) error {
	if err := opts.ComposeOpts.StopContainer(serviceName); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", serviceName, err)
	}
	
	if err := opts.ComposeOpts.RemoveContainer(serviceName); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", serviceName, err)
	}
	
	if err := opts.ComposeOpts.StartContainer(serviceName); err != nil {
		return fmt.Errorf("failed to start container %s: %w", serviceName, err)
	}
	
	return nil
}

// UpdateContainer checks if a container needs updating and updates if necessary
func (opts *UpdaterOptions) UpdateContainer(serviceName string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Updating %s", serviceName)
	s.Start()
	
	// Get current container ID
	currentContainerID, err := opts.ComposeOpts.GetCurrentContainerId(serviceName)
	if err != nil {
		s.FinalMSG = fmt.Sprintf("❌ Failed to get container ID for %s\n", serviceName)
		s.Stop()
		return fmt.Errorf("failed to get container ID for %s: %w", serviceName, err)
	}
	
	if currentContainerID == "" {
		opts.warnIfEnabled(s, fmt.Sprintf("%s is not running", serviceName))
		return nil
	}
	
	// Pull latest image first
	if err := opts.ComposeOpts.PullContainer(serviceName); err != nil {
		s.FinalMSG = fmt.Sprintf("❌ Failed to pull latest image for %s\n", serviceName)
		s.Stop()
		return fmt.Errorf("failed to pull latest image for %s: %w", serviceName, err)
	}
	
	// Get current and latest image IDs
	currentImageID, err := opts.DockerClient.GetCurrentImageId(currentContainerID)
	if err != nil {
		s.FinalMSG = fmt.Sprintf("❌ Failed to get current image ID for %s\n", serviceName)
		s.Stop()
		return fmt.Errorf("failed to get current image ID for %s: %w", serviceName, err)
	}
	
	latestImageID, err := opts.DockerClient.GetLatestImageId(currentContainerID)
	if err != nil {
		s.FinalMSG = fmt.Sprintf("❌ Failed to get latest image ID for %s\n", serviceName)
		s.Stop()
		return fmt.Errorf("failed to get latest image ID for %s: %w", serviceName, err)
	}
	
	// Compare image IDs and update if different
	if latestImageID != "" && currentImageID != latestImageID {
		s.Suffix = fmt.Sprintf(" Updating and restarting %s", serviceName)
		
		if err := opts.RestartContainer(serviceName); err != nil {
			s.FinalMSG = fmt.Sprintf("❌ Failed to restart %s\n", serviceName)
			s.Stop()
			return err
		}
		
		s.FinalMSG = fmt.Sprintf("✅ Updated %s\n", serviceName)
		s.Stop()
	} else {
		s.FinalMSG = fmt.Sprintf("✅ %s is already up to date\n", serviceName)
		s.Stop()
	}
	
	return nil
}
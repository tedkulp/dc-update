package core

import (
	"fmt"
	"os"
	"sync"
	"time"

	"dc-update/internal/compose"
	"dc-update/internal/docker"

	"github.com/briandowns/spinner"
	"golang.org/x/term"
)

// UpdaterOptions holds configuration for the updater
type UpdaterOptions struct {
	ShowWarnings   bool
	UseSpinners    bool
	ComposeOpts    *compose.Options
	DockerClient   *docker.Client
}

// isInteractiveTerminal checks if we're running in an interactive terminal
func isInteractiveTerminal() bool {
	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	
	// Check if stderr is a terminal
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}
	
	// Check for CI/automation environment variables
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE",
	}
	
	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}
	
	return true
}

// NewUpdaterOptions creates new updater options
func NewUpdaterOptions(composeFile string, showWarnings bool, nonInteractive bool) (*UpdaterOptions, error) {
	composeOpts := compose.NewOptions(composeFile)
	
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	// Determine spinner usage: disable if explicitly non-interactive or if terminal is non-interactive
	useSpinners := !nonInteractive && isInteractiveTerminal()
	
	return &UpdaterOptions{
		ShowWarnings: showWarnings,
		UseSpinners:  useSpinners,
		ComposeOpts:  composeOpts,
		DockerClient: dockerClient,
	}, nil
}

// Close closes the Docker client connection
func (opts *UpdaterOptions) Close() error {
	return opts.DockerClient.Close()
}

// SpinnerWrapper wraps spinner functionality with interactive terminal detection
type SpinnerWrapper struct {
	spinner    *spinner.Spinner
	useSpinner bool
	prefix     string
}

// NewSpinnerWrapper creates a new spinner wrapper
func (opts *UpdaterOptions) NewSpinnerWrapper(message string) *SpinnerWrapper {
	sw := &SpinnerWrapper{
		useSpinner: opts.UseSpinners,
		prefix:     message,
	}
	
	if sw.useSpinner {
		sw.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sw.spinner.Suffix = fmt.Sprintf(" %s", message)
	} else {
		// For non-interactive environments, just print the message
		fmt.Printf("⏳ %s\n", message)
	}
	
	return sw
}

// Start starts the spinner or prints a message
func (sw *SpinnerWrapper) Start() {
	if sw.useSpinner && sw.spinner != nil {
		sw.spinner.Start()
	}
}

// UpdateSuffix updates the spinner message
func (sw *SpinnerWrapper) UpdateSuffix(message string) {
	if sw.useSpinner && sw.spinner != nil {
		sw.spinner.Suffix = fmt.Sprintf(" %s", message)
	} else {
		fmt.Printf("⏳ %s\n", message)
	}
}

// Stop stops the spinner and shows final message
func (sw *SpinnerWrapper) Stop(finalMessage string) {
	if sw.useSpinner && sw.spinner != nil {
		sw.spinner.FinalMSG = fmt.Sprintf("%s\n", finalMessage)
		sw.spinner.Stop()
	} else {
		fmt.Printf("%s\n", finalMessage)
	}
}

// warnIfEnabled shows warnings based on the ShowWarnings flag
func (opts *UpdaterOptions) warnIfEnabled(sw *SpinnerWrapper, message string) {
	if opts.ShowWarnings {
		sw.Stop(fmt.Sprintf("⚠️  %s", message))
	} else {
		sw.Stop("") // Stop without message
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
	sw := opts.NewSpinnerWrapper(fmt.Sprintf("Updating %s", serviceName))
	sw.Start()
	
	// First validate that the service exists in the compose file
	if err := opts.ComposeOpts.ValidateServiceExists(serviceName); err != nil {
		sw.Stop(fmt.Sprintf("❌ Service '%s' does not exist in docker-compose file", serviceName))
		return err
	}
	
	// Get current container ID
	currentContainerID, err := opts.ComposeOpts.GetCurrentContainerId(serviceName)
	if err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to get container ID for %s", serviceName))
		return fmt.Errorf("failed to get container ID for %s: %w", serviceName, err)
	}
	
	if currentContainerID == "" {
		opts.warnIfEnabled(sw, fmt.Sprintf("%s is not running", serviceName))
		return nil
	}
	
	// Get the expected image name from the docker-compose file
	expectedImageName, err := opts.ComposeOpts.GetServiceImageName(serviceName)
	if err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to get image name for %s", serviceName))
		return fmt.Errorf("failed to get image name for %s: %w", serviceName, err)
	}
	
	// Get current container's image ID
	currentImageID, err := opts.DockerClient.GetCurrentImageId(currentContainerID)
	if err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to get current image ID for %s", serviceName))
		return fmt.Errorf("failed to get current image ID for %s: %w", serviceName, err)
	}
	
	// Pull the expected image to ensure we have the latest version
	if err := opts.ComposeOpts.PullContainer(serviceName); err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to pull image for %s", serviceName))
		return fmt.Errorf("failed to pull image for %s: %w", serviceName, err)
	}
	
	// Refresh image cache after pull to ensure we see the latest images
	if err := opts.DockerClient.RefreshImageCache(); err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to refresh image cache for %s", serviceName))
		return fmt.Errorf("failed to refresh image cache for %s: %w", serviceName, err)
	}
	
	// Get the expected image ID after pulling
	expectedImageID, err := opts.DockerClient.GetImageId(expectedImageName)
	if err != nil {
		sw.Stop(fmt.Sprintf("❌ Failed to get expected image ID for %s", serviceName))
		return fmt.Errorf("failed to get expected image ID for %s: %w", serviceName, err)
	}
	
	// Compare image IDs and update if different
	if expectedImageID != "" && currentImageID != expectedImageID {
		sw.UpdateSuffix(fmt.Sprintf("Updating and restarting %s", serviceName))
		
		if err := opts.RestartContainer(serviceName); err != nil {
			sw.Stop(fmt.Sprintf("❌ Failed to restart %s", serviceName))
			return err
		}
		
		sw.Stop(fmt.Sprintf("✅ Updated %s", serviceName))
	} else {
		sw.Stop(fmt.Sprintf("✅ %s is already up to date", serviceName))
	}
	
	return nil
}

// UpdateContainersConcurrently processes multiple containers with controlled concurrency
func (opts *UpdaterOptions) UpdateContainersConcurrently(serviceNames []string) error {
	const maxConcurrency = 3 // Limit concurrent operations to avoid overwhelming Docker daemon
	
	// Use a semaphore pattern to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	
	for _, serviceName := range serviceNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Update the container
			if err := opts.UpdateContainer(name); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("error updating %s: %w", name, err))
				fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", name, err)
				mu.Unlock()
			}
		}(serviceName)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// Return the first error if any occurred
	if len(errors) > 0 {
		return errors[0]
	}
	
	return nil
}
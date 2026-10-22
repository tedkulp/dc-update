package core

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"dc-update/internal/compose"
	"dc-update/internal/docker"

	"golang.org/x/term"
)

// UpdaterOptions holds configuration for the updater
type UpdaterOptions struct {
	ShowWarnings   bool
	ComposeOpts    *compose.Options
	DockerClient   *docker.Client
	// Status is the shared live status line; nil when not using spinners
	Status         *StatusRenderer
	// skipped counts services whose warnings were hidden by ShowWarnings
	skipped        atomic.Int32
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
	composeOpts, err := compose.NewOptions(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create compose options: %w", err)
	}
	
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	// Determine spinner usage: disable if explicitly non-interactive or if terminal is non-interactive
	useSpinners := !nonInteractive && isInteractiveTerminal()
	
	opts := &UpdaterOptions{
		ShowWarnings: showWarnings,
		ComposeOpts:  composeOpts,
		DockerClient: dockerClient,
	}
	if useSpinners {
		opts.Status = NewStatusRenderer(os.Stdout, terminalWidth)
		// Route log output (e.g. compose debug lines) around the live line
		log.SetOutput(opts.Status.Above(os.Stderr))
	}

	return opts, nil
}

// terminalWidth returns the width of the terminal on stdout, or 0 if unknown
func terminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return width
}

// Close closes the Docker client connection
func (opts *UpdaterOptions) Close() error {
	return opts.DockerClient.Close()
}

// SpinnerWrapper is the per-service status handle. In interactive mode it
// reports into the updater's shared StatusRenderer; otherwise it prints plain
// lines.
type SpinnerWrapper struct {
	status      *StatusRenderer
	serviceName string
	id          int
}

// NewSpinnerWrapper creates a status handle for a service
func (opts *UpdaterOptions) NewSpinnerWrapper(serviceName string) *SpinnerWrapper {
	sw := &SpinnerWrapper{
		status:      opts.Status,
		serviceName: serviceName,
	}

	if sw.status == nil {
		// For non-interactive environments, just print the message
		fmt.Printf("⏳ Updating %s\n", serviceName)
	}

	return sw
}

// Start puts the service on the live status line
func (sw *SpinnerWrapper) Start() {
	if sw.status != nil {
		sw.id = sw.status.Add(sw.serviceName, "checking", fmt.Sprintf("Updating %s", sw.serviceName))
	}
}

// SetPhase shows the service's current phase on the live status line. In
// non-interactive mode it prints message instead. An empty message keeps the
// previous one and prints nothing.
func (sw *SpinnerWrapper) SetPhase(phase string, message string) {
	if sw.status != nil {
		sw.status.SetPhase(sw.id, phase, message)
	} else if message != "" {
		fmt.Printf("⏳ %s\n", message)
	}
}

// Stop takes the service off the live status line and shows its final message
func (sw *SpinnerWrapper) Stop(finalMessage string) {
	if sw.status != nil {
		sw.status.Finish(sw.id, finalMessage)
	} else {
		fmt.Printf("%s\n", finalMessage)
	}
}

// Drop takes the service off the live status line without a final message
func (sw *SpinnerWrapper) Drop() {
	if sw.status != nil {
		sw.status.Finish(sw.id, "")
	}
}

// warnIfEnabled shows warnings based on the ShowWarnings flag. Hidden warnings
// are counted and reported as one summary line at the end.
func (opts *UpdaterOptions) warnIfEnabled(sw *SpinnerWrapper, message string) {
	if opts.ShowWarnings {
		sw.Stop(fmt.Sprintf("⚠️  %s", message))
	} else {
		opts.skipped.Add(1)
		sw.Drop()
	}
}

// skippedSummary describes how many services had their warnings hidden
func skippedSummary(count int) string {
	switch count {
	case 0:
		return ""
	case 1:
		return "⚠️  1 service skipped (use --show-warnings for details)"
	default:
		return fmt.Sprintf("⚠️  %d services skipped (use --show-warnings for details)", count)
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
	sw := opts.NewSpinnerWrapper(serviceName)
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
	sw.SetPhase("pulling", "")
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
	if expectedImageID == "" {
		// A blank expectedImageID means the image name from docker compose config
		// was not found in the local Docker daemon (e.g. the name lacks a :latest
		// suffix, or the image hasn't been pulled yet). This should not silently
		// report "already up to date".
		opts.warnIfEnabled(sw, fmt.Sprintf("Expected image ID not found for %s — image may not be pulled yet or name lacks a tag", serviceName))
		return nil
	}

	if currentImageID != expectedImageID {
		sw.SetPhase("restarting", fmt.Sprintf("Updating and restarting %s", serviceName))
		
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
	var errOut io.Writer = os.Stderr
	if opts.Status != nil {
		errOut = opts.Status.Above(os.Stderr)
	}
	
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
				fmt.Fprintf(errOut, "Error updating %s: %v\n", name, err)
				mu.Unlock()
			}
		}(serviceName)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()

	if summary := skippedSummary(int(opts.skipped.Swap(0))); summary != "" {
		fmt.Println(summary)
	}
	
	// Return the first error if any occurred
	if len(errors) > 0 {
		return errors[0]
	}
	
	return nil
}
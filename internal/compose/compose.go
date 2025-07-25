package compose

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options holds configuration for docker-compose operations
type Options struct {
	ComposeFile string
	WorkingDir  string
}

// NewOptions creates docker-compose options from a compose file path
func NewOptions(composeFilePath string) *Options {
	absPath, _ := filepath.Abs(composeFilePath)
	return &Options{
		ComposeFile: filepath.Base(absPath),
		WorkingDir:  filepath.Dir(absPath),
	}
}

// GetServiceNames executes `docker compose config --services` and returns service names
func (opts *Options) GetServiceNames() ([]string, error) {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "config", "--services")
	cmd.Dir = opts.WorkingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse output - split by newlines and filter empty strings
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	services := make([]string, 0, len(lines))

	for _, line := range lines {
		service := strings.TrimSpace(line)
		if service != "" {
			services = append(services, service)
		}
	}

	return services, nil
}

// GetCurrentContainerId executes `docker compose ps -q [service_name]` and returns container ID
func (opts *Options) GetCurrentContainerId(serviceName string) (string, error) {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "ps", "-q", serviceName)
	cmd.Dir = opts.WorkingDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output - trim whitespace and return container ID
	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// ValidateServiceExists checks if a service exists in the docker-compose file
func (opts *Options) ValidateServiceExists(serviceName string) error {
	services, err := opts.GetServiceNames()
	if err != nil {
		return fmt.Errorf("failed to get service list: %w", err)
	}
	
	for _, service := range services {
		if service == serviceName {
			return nil
		}
	}
	
	return fmt.Errorf("service '%s' does not exist in docker-compose file", serviceName)
}

// StopContainer executes `docker compose stop [service]`
func (opts *Options) StopContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "stop", serviceName)
	cmd.Dir = opts.WorkingDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", serviceName, err)
	}
	return nil
}

// RemoveContainer executes `docker compose rm [service]`
func (opts *Options) RemoveContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "rm", "-f", serviceName)
	cmd.Dir = opts.WorkingDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container '%s': %w", serviceName, err)
	}
	return nil
}

// StartContainer executes `docker compose up -d [service]`
func (opts *Options) StartContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "up", "-d", serviceName)
	cmd.Dir = opts.WorkingDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container '%s': %w", serviceName, err)
	}
	return nil
}

// PullContainer executes `docker compose pull [service]`
func (opts *Options) PullContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "pull", serviceName)
	cmd.Dir = opts.WorkingDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image for '%s': %w", serviceName, err)
	}
	return nil
}

// BuildContainers executes `docker compose build --pull [services...]`
func (opts *Options) BuildContainers(serviceNames []string) error {
	args := []string{"compose", "-f", opts.ComposeFile, "build", "--pull"}
	args = append(args, serviceNames...)

	cmd := exec.Command("docker", args...)
	cmd.Dir = opts.WorkingDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build containers %v: %w", serviceNames, err)
	}
	return nil
}

// GetServiceImageName gets the image name for a specific service from docker-compose config
func (opts *Options) GetServiceImageName(serviceName string) (string, error) {
	// Verify service exists first
	if err := opts.ValidateServiceExists(serviceName); err != nil {
		return "", err
	}
	
	// Use docker compose config to get the YAML and parse it
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "config")
	cmd.Dir = opts.WorkingDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get docker-compose config: %w", err)
	}
	
	// Parse the YAML output to find the image for this service
	lines := strings.Split(string(output), "\n")
	inService := false
	serviceIndent := ""
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're entering our target service
		if strings.HasPrefix(trimmed, serviceName+":") {
			inService = true
			serviceIndent = strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
			continue
		}
		
		// If we're in the service, look for the image field
		if inService {
			// Check if we've moved to a different service (same or less indentation)
			currentIndent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
			if len(currentIndent) <= len(serviceIndent) && trimmed != "" && !strings.HasPrefix(line, serviceIndent+" ") {
				inService = false
				continue
			}
			
			// Look for image: field
			if strings.HasPrefix(trimmed, "image:") {
				imageName := strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
				// Remove quotes if present
				imageName = strings.Trim(imageName, "\"'")
				return imageName, nil
			}
		}
	}
	
	return "", fmt.Errorf("could not find image for service %s", serviceName)
}

// RestartContainer is a convenience method that stops, removes, and starts a container
func (opts *Options) RestartContainer(serviceName string) error {
	if err := opts.StopContainer(serviceName); err != nil {
		return err
	}

	if err := opts.RemoveContainer(serviceName); err != nil {
		return err
	}

	return opts.StartContainer(serviceName)
}

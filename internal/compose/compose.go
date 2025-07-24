package compose

import (
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

// StopContainer executes `docker compose stop [service]`
func (opts *Options) StopContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "stop", serviceName)
	cmd.Dir = opts.WorkingDir
	return cmd.Run()
}

// RemoveContainer executes `docker compose rm [service]`
func (opts *Options) RemoveContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "rm", "-f", serviceName)
	cmd.Dir = opts.WorkingDir
	return cmd.Run()
}

// StartContainer executes `docker compose up -d [service]`
func (opts *Options) StartContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "up", "-d", serviceName)
	cmd.Dir = opts.WorkingDir
	return cmd.Run()
}

// PullContainer executes `docker compose pull [service]`
func (opts *Options) PullContainer(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", opts.ComposeFile, "pull", serviceName)
	cmd.Dir = opts.WorkingDir
	return cmd.Run()
}

// BuildContainers executes `docker compose build --pull [services...]`
func (opts *Options) BuildContainers(serviceNames []string) error {
	args := []string{"compose", "-f", opts.ComposeFile, "build", "--pull"}
	args = append(args, serviceNames...)
	
	cmd := exec.Command("docker", args...)
	cmd.Dir = opts.WorkingDir
	return cmd.Run()
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
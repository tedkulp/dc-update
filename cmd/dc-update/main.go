package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dc-update/internal/core"

	"github.com/urfave/cli/v2"
)

// Version information set by GoReleaser
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	app := &cli.App{
		Name:  "dc-update",
		Version: version,
		Usage: "An opinionated script for updating large docker-compose based systems",
		UsageText: "dc-update [CONTAINER_NAME]...",
		Description: `dc-update intelligently updates only containers that have newer images available, avoiding unnecessary restarts.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Path to docker-compose.yml file",
				Value:   "docker-compose.yml",
			},
			&cli.StringSliceFlag{
				Name:    "build",
				Aliases: []string{"b"},
				Usage:   "Container to build before updating. Can be called multiple times",
			},
			&cli.BoolFlag{
				Name:  "show-warnings",
				Usage: "Show warnings for containers that aren't running (default: false)",
			},
			&cli.BoolFlag{
				Name:    "non-interactive",
				Aliases: []string{"n"},
				Usage:   "Disable spinners and use plain text output",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// Validate docker-compose file existence
			dockerComposeFile := cCtx.String("file")
			if !filepath.IsAbs(dockerComposeFile) {
				dockerComposeFile = filepath.Join(".", dockerComposeFile)
			}
			
			if _, err := os.Stat(dockerComposeFile); os.IsNotExist(err) {
				return fmt.Errorf("docker-compose file does not exist: %s", dockerComposeFile)
			}

			// Get CLI arguments
			containerNames := cCtx.Args().Slice()
			buildContainers := cCtx.StringSlice("build")
			showWarnings := cCtx.Bool("show-warnings")
			nonInteractive := cCtx.Bool("non-interactive")

			// Initialize updater
			updater, err := core.NewUpdaterOptions(dockerComposeFile, showWarnings, nonInteractive)
			if err != nil {
				return fmt.Errorf("failed to initialize updater: %w", err)
			}
			defer updater.Close()

			// Determine service names - use args if provided, otherwise get all services
			var serviceNames []string
			if len(containerNames) > 0 {
				serviceNames = containerNames
			} else {
				allServices, err := updater.ComposeOpts.GetServiceNames()
				if err != nil {
					return fmt.Errorf("failed to get service names: %w", err)
				}
				serviceNames = allServices
			}

			// Handle build containers if specified
			if len(buildContainers) > 0 {
				fmt.Printf("Building containers: %v\n", buildContainers)
				if err := updater.ComposeOpts.BuildContainers(buildContainers); err != nil {
					return fmt.Errorf("failed to build containers: %w", err)
				}
			}

			// Process containers with controlled concurrency
			if err := updater.UpdateContainersConcurrently(serviceNames); err != nil {
				return fmt.Errorf("failed to update containers: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
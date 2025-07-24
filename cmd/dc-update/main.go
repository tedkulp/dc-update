package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "dc-update",
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

			// Get container names from arguments
			containerNames := cCtx.Args().Slice()
			buildContainers := cCtx.StringSlice("build")
			showWarnings := cCtx.Bool("show-warnings")

			// Debug output for now
			fmt.Printf("Docker-compose file: %s\n", dockerComposeFile)
			fmt.Printf("Container names: %v\n", containerNames)
			fmt.Printf("Build containers: %v\n", buildContainers)
			fmt.Printf("Show warnings: %v\n", showWarnings)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
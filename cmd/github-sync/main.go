// Package main provides the entry point for the gh-mirror application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"i2pgit.org/go-i2p/go-github-sync/pkg/config"
	"i2pgit.org/go-i2p/go-github-sync/pkg/git"
	"i2pgit.org/go-i2p/go-github-sync/pkg/github"
	"i2pgit.org/go-i2p/go-github-sync/pkg/logger"
	"i2pgit.org/go-i2p/go-github-sync/pkg/workflow"
	"github.com/spf13/cobra"
)

func main() {
	log := logger.New(false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Received termination signal, shutting down...")
		cancel()
		os.Exit(1)
	}()

	rootCmd := &cobra.Command{
		Use:   "gh-mirror",
		Short: "GitHub Mirror Sync Tool",
		Long:  "Tool for generating GitHub Actions workflow to sync external repositories to GitHub mirrors",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(ctx, log)
		},
	}

	// Add flags
	config.AddFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	// Parse configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	log.Info("Configuration loaded successfully",
		"primary_repo", cfg.PrimaryRepo,
		"mirror_repo", cfg.MirrorRepo,
		"primary_branch", cfg.PrimaryBranch,
		"mirror_branch", cfg.MirrorBranch,
		"sync_interval", cfg.SyncInterval)

	// Update logger verbosity if needed
	if cfg.Verbose {
		log = logger.New(true)
	}

	// Validate Git repositories
	gitClient := git.NewClient(log)
	err = gitClient.ValidateRepos(ctx, cfg)
	if err != nil {
		return fmt.Errorf("repository validation failed: %w", err)
	}
	log.Info("Git repositories validated successfully")

	// Setup GitHub client
	githubClient, err := github.NewClient(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}
	log.Info("GitHub client initialized successfully")

	// Generate workflow file
	generator := workflow.NewGenerator(cfg, log)
	workflowYAML, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate workflow file: %w", err)
	}
	log.Info("Workflow file generated successfully")

	// Setup GitHub repository (optional)
	if cfg.SetupWorkflow {
		err = githubClient.SetupWorkflow(ctx, workflowYAML)
		if err != nil {
			return fmt.Errorf("failed to setup GitHub workflow: %w", err)
		}
		log.Info("GitHub workflow set up successfully")
	} else {
		// Write workflow to stdout or file
		if cfg.OutputFile != "" {
			err = os.WriteFile(cfg.OutputFile, []byte(workflowYAML), 0644)
			if err != nil {
				return fmt.Errorf("failed to write workflow to file: %w", err)
			}
			log.Info("Workflow written to file", "file", cfg.OutputFile)
		} else {
			fmt.Println(workflowYAML)
		}
	}

	return nil
}

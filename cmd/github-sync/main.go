// Package main provides the entry point for the gh-mirror application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-i2p/go-github-sync/pkg/config"
	"github.com/go-i2p/go-github-sync/pkg/git"
	"github.com/go-i2p/go-github-sync/pkg/github"
	"github.com/go-i2p/go-github-sync/pkg/logger"
	"github.com/go-i2p/go-github-sync/pkg/workflow"
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

	// Auto-detect GitHub remote if mirror is not specified
	if cfg.MirrorRepo == "" {
		mirrorRepo, err := detectGithubRemote(ctx)
		if err != nil {
			log.Debug("Failed to auto-detect GitHub remote", "error", err)
		} else if mirrorRepo != "" {
			cfg.MirrorRepo = mirrorRepo
			log.Info("Auto-detected GitHub mirror repository", "mirror_repo", cfg.MirrorRepo)
		}
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

// detectGithubRemote attempts to detect a GitHub remote URL from the current git repository
func detectGithubRemote(ctx context.Context) (string, error) {
	// Execute git remote -v command
	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute git remote command: %w", err)
	}

	// Parse the output to find GitHub remotes
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "github.com") && strings.Contains(line, "(push)") {
			// Extract the GitHub repository URL
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				url := parts[1]
				// Convert SSH URL to HTTPS URL if needed
				if strings.HasPrefix(url, "git@github.com:") {
					url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
				}
				// Remove .git suffix if present
				url = strings.TrimSuffix(url, ".git")
				return url, nil
			}
		}
	}

	return "", fmt.Errorf("no GitHub remote found")
}

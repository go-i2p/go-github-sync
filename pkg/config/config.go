// Package config handles the configuration settings for the GitHub mirror sync tool.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Config holds the application configuration.
type Config struct {
	// GitHub token for authentication
	GithubToken string

	// Repository URLs
	PrimaryRepo string
	MirrorRepo  string

	// Branch names
	PrimaryBranch string
	MirrorBranch  string

	// Synchronization settings
	SyncInterval string
	ForceSync    bool

	// Output configuration
	OutputFile    string
	SetupWorkflow bool
	Verbose       bool
}

var (
	config Config

	// Flags
	primaryRepo   string
	mirrorRepo    string
	primaryBranch string
	mirrorBranch  string
	syncInterval  string
	forceSync     bool
	outputFile    string
	setupWorkflow bool
	verbose       bool
)

// AddFlags adds the configuration flags to the given command.
func AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&primaryRepo, "primary", "p", "", "Primary repository URL (required)")
	cmd.Flags().StringVarP(&mirrorRepo, "mirror", "m", "", "GitHub mirror repository URL (required)")
	cmd.Flags().StringVar(&primaryBranch, "primary-branch", "main", "Primary repository branch name")
	cmd.Flags().StringVar(&mirrorBranch, "mirror-branch", "main", "GitHub mirror repository branch name")
	cmd.Flags().StringVarP(&syncInterval, "interval", "i", "hourly", "Sync interval (hourly, daily, weekly)")
	cmd.Flags().BoolVar(&forceSync, "force", true, "Force sync by overwriting mirror with primary content")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for workflow YAML (writes to stdout if not specified)")
	cmd.Flags().BoolVar(&setupWorkflow, "setup", false, "Automatically setup the workflow in the GitHub repository")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	cmd.MarkFlagRequired("primary")
	cmd.MarkFlagRequired("mirror")
}

// Load parses the flags and environment variables to build the configuration.
func Load() (*Config, error) {
	// Get GitHub token from environment
	githubToken := os.Getenv("GH_TOKEN")
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" && setupWorkflow {
		return nil, fmt.Errorf("GitHub token not found in environment (GH_TOKEN or GITHUB_TOKEN) but required for --setup")
	}

	// Validate repositories
	if primaryRepo == "" {
		return nil, fmt.Errorf("primary repository URL is required")
	}
	if mirrorRepo == "" {
		return nil, fmt.Errorf("mirror repository URL is required")
	}

	// Validate sync interval
	switch strings.ToLower(syncInterval) {
	case "hourly", "daily", "weekly":
		// valid
	default:
		return nil, fmt.Errorf("invalid sync interval: %s (must be hourly, daily, or weekly)", syncInterval)
	}

	// Set the values in the config struct
	config = Config{
		GithubToken:   githubToken,
		PrimaryRepo:   primaryRepo,
		MirrorRepo:    mirrorRepo,
		PrimaryBranch: primaryBranch,
		MirrorBranch:  mirrorBranch,
		SyncInterval:  syncInterval,
		ForceSync:     forceSync,
		OutputFile:    outputFile,
		SetupWorkflow: setupWorkflow,
		Verbose:       verbose,
	}

	return &config, nil
}

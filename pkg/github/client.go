// Package github provides functionality for interacting with the GitHub API.
package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"

	"github.com/go-i2p/go-gh-mirror/pkg/config"
	"github.com/go-i2p/go-gh-mirror/pkg/logger"
)

const (
	workflowPath = ".github/workflows/sync-mirror.yml"
)

// Client provides GitHub API functionality.
type Client struct {
	client *github.Client
	log    *logger.Logger
	cfg    *config.Config
	owner  string
	repo   string
}

// NewClient creates a new GitHub API client.
func NewClient(ctx context.Context, cfg *config.Config, log *logger.Logger) (*Client, error) {
	var httpClient *http.Client

	// Create authenticated client if token is available
	if cfg.GithubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.GithubToken},
		)
		httpClient = oauth2.NewClient(ctx, ts)
		log.Debug("Created authenticated GitHub client")
	} else {
		httpClient = http.DefaultClient
		log.Debug("Created unauthenticated GitHub client")
	}

	// Create GitHub client
	client := github.NewClient(httpClient)

	// Parse owner and repo from mirror URL
	owner, repo, err := parseGitHubURL(cfg.MirrorRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub repository URL: %w", err)
	}

	return &Client{
		client: client,
		log:    log,
		cfg:    cfg,
		owner:  owner,
		repo:   repo,
	}, nil
}

// SetupWorkflow creates or updates the workflow file in the repository.
func (c *Client) SetupWorkflow(ctx context.Context, workflowContent string) error {
	c.log.Info("Setting up workflow in repository", "owner", c.owner, "repo", c.repo, "path", workflowPath)

	// Check if the file already exists
	fileContent, _, resp, err := c.client.Repositories.GetContents(
		ctx,
		c.owner,
		c.repo,
		workflowPath,
		&github.RepositoryContentGetOptions{},
	)

	// Create a commit message based on whether we're creating or updating
	commitMsg := "Add repository sync workflow"
	var sha *string

	if err == nil && resp.StatusCode == http.StatusOK && fileContent != nil {
		// File exists, we'll update it
		commitMsg = "Update repository sync workflow"
		sha = fileContent.SHA
		c.log.Debug("Updating existing workflow file", "sha", *sha)
	} else if resp != nil && resp.StatusCode != http.StatusNotFound {
		// Unexpected error
		return fmt.Errorf("failed to check for existing workflow file: %w", err)
	}

	// Create or update the file
	_, _, err = c.client.Repositories.CreateFile(
		ctx,
		c.owner,
		c.repo,
		workflowPath,
		&github.RepositoryContentFileOptions{
			Message: &commitMsg,
			Content: []byte(workflowContent),
			SHA:     sha,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to create/update workflow file: %w", err)
	}

	c.log.Info("Workflow file successfully created/updated")
	return nil
}

// parseGitHubURL extracts the owner and repository from a GitHub URL.
func parseGitHubURL(githubURL string) (string, string, error) {
	// Handle HTTP(S) URLs
	if strings.HasPrefix(githubURL, "http://") || strings.HasPrefix(githubURL, "https://") {
		parsedURL, err := url.Parse(githubURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		pathParts := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub repository path: %s", parsedURL.Path)
		}
		return pathParts[0], strings.TrimSuffix(pathParts[1], ".git"), nil
	}

	// Handle SSH URLs
	if strings.HasPrefix(githubURL, "git@github.com:") {
		path := strings.TrimPrefix(githubURL, "git@github.com:")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL format")
		}
		return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
	}

	return "", "", fmt.Errorf("unsupported GitHub URL format")
}

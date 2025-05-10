// Package git provides Git-related operations and validation.
package git

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-i2p/go-github-sync/pkg/config"
	"github.com/go-i2p/go-github-sync/pkg/logger"
)

// Client provides Git repository validation and operations.
type Client struct {
	log        *logger.Logger
	httpClient *http.Client
}

// NewClient creates a new Git client.
func NewClient(log *logger.Logger) *Client {
	return &Client{
		log: log,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateRepos checks if both repositories are accessible.
func (c *Client) ValidateRepos(ctx context.Context, cfg *config.Config) error {
	// Validate primary repository URL
	if err := c.validateRepoURL(ctx, cfg.PrimaryRepo); err != nil {
		return fmt.Errorf("invalid primary repository URL: %w", err)
	}

	// Validate GitHub repository URL format
	if !strings.Contains(cfg.MirrorRepo, "github.com") {
		return fmt.Errorf("mirror repository must be a GitHub repository URL")
	}

	// Extract owner and repo from GitHub URL
	owner, repo, err := parseGitHubURL(cfg.MirrorRepo)
	if err != nil {
		return fmt.Errorf("failed to parse GitHub repository URL: %w", err)
	}

	c.log.Debug("Parsed GitHub repository", "owner", owner, "repo", repo)
	return nil
}

// validateRepoURL checks if a Git repository URL is accessible.
func (c *Client) validateRepoURL(ctx context.Context, repoURL string) error {
	// For HTTP/HTTPS URLs, try to access the repository
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		// For GitHub URLs, we can check info/refs
		if strings.Contains(repoURL, "github.com") {
			checkURL := ensureGitExtension(repoURL) + "/info/refs?service=git-upload-pack"
			return c.checkEndpoint(ctx, checkURL)
		}

		// For other Git servers, just try a HEAD request on the base URL
		return c.checkEndpoint(ctx, ensureGitExtension(repoURL))
	}

	// For SSH URLs, we can't easily validate, so just check the format
	if strings.HasPrefix(repoURL, "git@") || strings.HasPrefix(repoURL, "ssh://") {
		// Basic validation for SSH URLs
		if !strings.Contains(repoURL, ":") && !strings.Contains(repoURL, "/") {
			return fmt.Errorf("invalid SSH URL format")
		}
		c.log.Debug("SSH URL provided, cannot fully validate accessibility", "url", repoURL)
		return nil
	}

	return fmt.Errorf("unsupported repository URL scheme")
}

// checkEndpoint makes a HEAD request to check if an endpoint is accessible.
func (c *Client) checkEndpoint(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to access repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("repository returned error status: %s", resp.Status)
	}

	return nil
}

// parseGitHubURL extracts the owner and repository from a GitHub URL.
func parseGitHubURL(githubURL string) (string, string, error) {
	// Clean the URL to ensure we have the correct format
	cleanURL := ensureGitExtension(githubURL)

	// Parse the URL
	parsedURL, err := url.Parse(cleanURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Handle HTTP(S) URLs
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
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

// ensureGitExtension ensures the URL ends with .git for Git operations.
func ensureGitExtension(repoURL string) string {
	if !strings.HasSuffix(repoURL, ".git") {
		return repoURL + ".git"
	}
	return repoURL
}

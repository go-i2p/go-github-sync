# go-github-sync

A Go tool that generates and sets up GitHub Actions workflows to automatically sync external repositories to GitHub mirrors. This tool was created to make the life of a git admin easier.

## Overview

The `go-github-sync` tool creates GitHub Actions workflows that periodically pull changes from a primary repository and push them to a GitHub mirror repository. It can either output the workflow YAML file or directly set it up in the target GitHub repository.

## Installation

```bash
# Clone the repository
git clone https://i2pgit.org/go-i2p/go-github-sync.git

# Build the tool
cd go-github-sync
go build -o github-sync ./cmd/github-sync
```

## Usage

```bash
# Basic usage
github-sync --primary https://example.org/repo.git --mirror https://github.com/user/repo

# Output workflow to file
github-sync --primary https://example.org/repo.git --mirror https://github.com/user/repo --output workflow.yml

# Setup workflow in GitHub repository
github-sync --primary https://example.org/repo.git --mirror https://github.com/user/repo --setup
```

### Command Line Options

- `--primary`, `-p`: Primary repository URL (required)
- `--mirror`, `-m`: GitHub mirror repository URL (required, auto-detected if possible)
- `--primary-branch`: Primary repository branch name (default: "main")
- `--mirror-branch`: GitHub mirror repository branch name (default: "main")
- `--interval`, `-i`: Sync interval - hourly, daily, weekly (default: "hourly")
- `--force`: Force sync by overwriting mirror with primary content (default: true)
- `--output`, `-o`: Output file for workflow YAML (default: ".github/workflows/sync.yaml")
- `--setup`: Automatically setup the workflow in the GitHub repository
- `--verbose`, `-v`: Enable verbose logging

## Requirements

- GitHub token (needed when using `--setup` flag)
  - Set via `GITHUB_TOKEN` or `GH_TOKEN` environment variable

## Dependencies

- github.com/google/go-github/v61
- github.com/spf13/cobra
- go.uber.org/zap
- golang.org/x/oauth2
- gopkg.in/yaml.v3

## License

MIT License
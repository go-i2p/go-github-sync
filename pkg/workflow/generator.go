// Package workflow generates the GitHub Actions workflow file.
package workflow

import (
	"bytes"
	"fmt"
	"text/template"

	"gopkg.in/yaml.v3"

	"i2pgit.org/go-i2p/go-github-sync/pkg/config"
	"i2pgit.org/go-i2p/go-github-sync/pkg/logger"
)

// Generator generates GitHub Actions workflow files.
type Generator struct {
	cfg *config.Config
	log *logger.Logger
}

// WorkflowTemplate is the structure for the GitHub Actions workflow.
type WorkflowTemplate struct {
	PrimaryRepo   string
	MirrorRepo    string
	PrimaryBranch string
	MirrorBranch  string
	CronSchedule  string
	ForceSync     bool
}

// NewGenerator creates a new workflow generator.
func NewGenerator(cfg *config.Config, log *logger.Logger) *Generator {
	return &Generator{
		cfg: cfg,
		log: log,
	}
}

// Generate creates a GitHub Actions workflow YAML file.
func (g *Generator) Generate() (string, error) {
	// Determine cron schedule based on sync interval
	cronSchedule := getCronSchedule(g.cfg.SyncInterval)
	g.log.Debug("Using cron schedule", "schedule", cronSchedule)

	// Prepare template data
	data := WorkflowTemplate{
		PrimaryRepo:   g.cfg.PrimaryRepo,
		MirrorRepo:    g.cfg.MirrorRepo,
		PrimaryBranch: g.cfg.PrimaryBranch,
		MirrorBranch:  g.cfg.MirrorBranch,
		CronSchedule:  cronSchedule,
		ForceSync:     g.cfg.ForceSync,
	}

	// Generate workflow file from template
	workflowYAML, err := generateWorkflowYAML(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate workflow YAML: %w", err)
	}

	return workflowYAML, nil
}

// getCronSchedule converts a sync interval to a cron schedule.
func getCronSchedule(interval string) string {
	switch interval {
	case "hourly":
		return "0 * * * *"
	case "daily":
		return "0 0 * * *"
	case "weekly":
		return "0 0 * * 0"
	default:
		return "0 * * * *" // Default to hourly
	}
}

// generateWorkflowYAML creates the complete workflow YAML from the template.
func generateWorkflowYAML(data WorkflowTemplate) (string, error) {
	// Create the workflow structure using maps to maintain comment ordering
	workflow := map[string]interface{}{
		"name": "Sync Primary Repository to GitHub Mirror",
		"on": map[string]interface{}{
			"push": map[string]interface{}{},
			"schedule": []map[string]string{
				{"cron": data.CronSchedule},
			},
			"workflow_dispatch": map[string]interface{}{}, // Allow manual triggering
		},
		"jobs": map[string]interface{}{
			"sync": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"steps": []map[string]interface{}{
					{
						"name": "Validate Github Actions Environment",
						"run":  "if [ \"$GITHUB_ACTIONS\" != \"true\" ]; then echo 'This script must be run in a GitHub Actions environment.'; exit 1; fi",
					},
					{
						"name": "Checkout GitHub Mirror",
						"uses": "actions/checkout@v3",
						"with": map[string]interface{}{
							"fetch-depth": 0,
						},
					},
					{
						"name": "Configure Git",
						"run":  "git config user.name 'GitHub Actions'\ngit config user.email 'actions@github.com'",
					},
					{
						"name": "Sync Primary Repository",
						"run":  generateSyncScript(data),
						"env": map[string]string{
							"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
						},
					},
				},
			},
		},
	}

	// Convert workflow to YAML
	var buf bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&buf)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to encode workflow to YAML: %w", err)
	}

	// Add comments to the generated YAML
	result := addComments(buf.String())
	return result, nil
}

// generateSyncScript creates the Git commands for syncing repositories.
func generateSyncScript(data WorkflowTemplate) string {
	tmpl := `# Add the primary repository as a remote
git remote add primary {{.PrimaryRepo}}

# Fetch the latest changes from the primary repository
git fetch primary

# Check if the primary branch exists in the primary repository
if git ls-remote --heads primary {{.PrimaryBranch}} | grep -q {{.PrimaryBranch}}; then
  echo "Primary branch {{.PrimaryBranch}} found in primary repository"
else
  echo "Error: Primary branch {{.PrimaryBranch}} not found in primary repository"
  exit 1
fi

# Check if we're already on the mirror branch
if git rev-parse --verify --quiet {{.MirrorBranch}}; then
  git checkout {{.MirrorBranch}}
else
  # Create the mirror branch if it doesn't exist
  git checkout -b {{.MirrorBranch}}
fi

{{if .ForceSync}}
# Force-apply all changes from primary, overriding any conflicts
echo "Performing force sync from primary/{{.PrimaryBranch}} to {{.MirrorBranch}}"
git reset --hard primary/{{.PrimaryBranch}}
{{else}}
# Attempt to merge changes from primary
echo "Attempting to merge changes from primary/{{.PrimaryBranch}} to {{.MirrorBranch}}"
if ! git merge primary/{{.PrimaryBranch}} --no-edit; then
  # If merge fails, prefer the primary repository's changes
  echo "Merge conflict detected, preferring primary repository's changes"
  git checkout --theirs .
  git add .
  git commit -m "Merge primary repository, preferring primary changes in conflicts"
fi
{{end}}

# Push changes back to the mirror repository
git push origin {{.MirrorBranch}}`

	t, err := template.New("sync").Parse(tmpl)
	if err != nil {
		return "echo 'Error generating sync script'" // Fallback
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "echo 'Error generating sync script'" // Fallback
	}

	return buf.String()
}

// addComments adds explanatory comments to the YAML.
func addComments(yaml string) string {
	header := `# GitHub Actions workflow file to sync an external repository to this GitHub mirror.
# This file was automatically generated by go-github-sync.
#
# The workflow does the following:
# - Runs on a scheduled basis (and can also be triggered manually)
# - Clones the GitHub mirror repository
# - Fetches changes from the primary external repository
# - Applies those changes to the mirror repository
# - Pushes the updated content back to the GitHub mirror
#
# Authentication is handled by the GITHUB_TOKEN secret provided by GitHub Actions.

`
	return header + yaml
}

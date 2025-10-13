package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v74/github"
)

// terraform workflows including:
// Template rendering
// Terraform execution
// GitOps integration
type Orchestrator struct {
	executor       *Executor
	templateBaseDir string
}

type OrchestratorConfig struct {
	WorkDir         string
	TemplateBaseDir string
	EnvVars         map[string]string
}

func NewOrchestrator(config OrchestratorConfig) (*Orchestrator, error) {
	executor, err := NewExecutor(config.WorkDir, config.EnvVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return &Orchestrator{
		executor:        executor,
		templateBaseDir: config.TemplateBaseDir,
	}, nil
}

// loads a template file and renders it with the given data
func (o *Orchestrator) RenderTemplate(templatePath string, data any) (string, error) {
	fullPath := filepath.Join(o.templateBaseDir, templatePath)

	tmpl, err := template.ParseFiles(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.String(), nil
}

// renders a template and writes it to a file in the work directory
func (o *Orchestrator) RenderTemplateToFile(templatePath string, outputFilename string, data any) error {
	content, err := o.RenderTemplate(templatePath, data)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(o.executor.WorkDir(), outputFilename)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

func (o *Orchestrator) Init(ctx context.Context) error {
	return o.executor.Init(ctx)
}

func (o *Orchestrator) Plan(ctx context.Context) (bool, error) {
	return o.executor.Plan(ctx)
}

func (o *Orchestrator) Apply(ctx context.Context) error {
	return o.executor.Apply(ctx)
}

func (o *Orchestrator) Destroy(ctx context.Context) error {
	return o.executor.Destroy(ctx)
}

func (o *Orchestrator) ForceUnlock(ctx context.Context, lockID string) error {
	return o.executor.ForceUnlock(ctx, lockID)
}

func (o *Orchestrator) Output(ctx context.Context) (map[string]interface{}, error) {
	outputMeta, err := o.executor.Output(ctx)
	if err != nil {
		return nil, err
	}

	// Convert OutputMeta to simple map
	result := make(map[string]interface{})
	for key, meta := range outputMeta {
		result[key] = meta.Value
	}

	return result, nil
}

type GitOpsConfig struct {
	Owner    string
	Repo     string
	Branch   string
	BasePath string
	Username string
	Email    string
}

func (o *Orchestrator) CommitToGitOps(ctx context.Context, ghClient *github.Client, config GitOpsConfig, commitMessage string) error {
	// Get the latest commit SHA for the branch (we refresh this before committing)
	ref, _, err := ghClient.Git.GetRef(ctx, config.Owner, config.Repo, "refs/heads/"+config.Branch)
	if err != nil {
		return fmt.Errorf("failed to get branch ref: %w", err)
	}
	baseCommitSHA := ref.GetObject().GetSHA()

	// Get the base tree SHA
	baseCommit, _, err := ghClient.Git.GetCommit(ctx, config.Owner, config.Repo, baseCommitSHA)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}
	baseTreeSHA := baseCommit.GetTree().GetSHA()

	// Read all .tf files from work directory and create tree entries
	var treeEntries []*github.TreeEntry
	err = filepath.Walk(o.executor.WorkDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".tf" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Get relative path from workDir
		relPath, err := filepath.Rel(o.executor.WorkDir(), path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create blob for file content
		blob, _, err := ghClient.Git.CreateBlob(ctx, config.Owner, config.Repo, &github.Blob{
			Content:  github.Ptr(string(content)),
			Encoding: github.Ptr("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %w", relPath, err)
		}

		// Add to tree with path in configured base path
		gitPath := filepath.Join(config.BasePath, relPath)
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(gitPath),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  blob.SHA,
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to process terraform files: %w", err)
	}

	if len(treeEntries) == 0 {
		return fmt.Errorf("no .tf files found in %s", o.executor.WorkDir())
	}

	// Create a new tree with the terraform files
	tree, _, err := ghClient.Git.CreateTree(ctx, config.Owner, config.Repo, baseTreeSHA, treeEntries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Check if tree is identical to base tree (no changes)
	if tree.GetSHA() == baseTreeSHA {
		fmt.Println("No changes detected in tree - skipping commit")
		return nil
	}

	// Refresh the branch ref to get the latest commit (for concurrent changes)
	latestRef, _, err := ghClient.Git.GetRef(ctx, config.Owner, config.Repo, "refs/heads/"+config.Branch)
	if err != nil {
		return fmt.Errorf("failed to refresh branch ref: %w", err)
	}
	latestCommitSHA := latestRef.GetObject().GetSHA()

	// If the branch moved since we started, use the latest commit as parent
	parentSHA := baseCommitSHA
	if latestCommitSHA != baseCommitSHA {
		fmt.Printf("Branch moved during operation (was %s, now %s) - using latest as parent\n",
			baseCommitSHA[:7], latestCommitSHA[:7])
		parentSHA = latestCommitSHA
	}

	// Create commit
	now := time.Now()
	author := &github.CommitAuthor{
		Name:  github.Ptr(config.Username),
		Email: github.Ptr(config.Email),
		Date:  &github.Timestamp{Time: now},
	}

	commit, _, err := ghClient.Git.CreateCommit(ctx, config.Owner, config.Repo, &github.Commit{
		Message:   github.Ptr(commitMessage),
		Tree:      tree,
		Parents:   []*github.Commit{{SHA: github.Ptr(parentSHA)}},
		Author:    author,
		Committer: author,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Update branch reference to point to new commit
	latestRef.Object.SHA = commit.SHA
	_, _, err = ghClient.Git.UpdateRef(ctx, config.Owner, config.Repo, latestRef, false)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	fmt.Printf("Successfully committed to %s/%s (branch: %s)\n", config.Owner, config.Repo, config.Branch)
	return nil
}

func (o *Orchestrator) WorkDir() string {
	return o.executor.WorkDir()
}

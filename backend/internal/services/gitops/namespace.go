package gitops

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/stolos-cloud/stolos/backend/internal/models"
)

// CreateNamespaceDirectory creates a directory in deployments/ for the namespace
func (s *GitOpsService) CreateNamespaceDirectory(ctx context.Context, namespaceName string) error {
	ghClient, err := s.GetGitHubClient()
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	owner, repo := ghClient.GetRepoInfo()
	branch := ghClient.GetRepoBranch()

	gitopsConfig, err := s.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Create namespace directory in deployments/ with a .gitkeep file
	files := map[string]string{
		fmt.Sprintf("deployments/%s/.gitkeep", namespaceName): "",
	}

	commitMsg := fmt.Sprintf("Create namespace %s", namespaceName)
	if err := s.commitFilesToGitHub(ctx, ghClient.Client, owner, repo, branch, files, commitMsg, gitopsConfig); err != nil {
		return fmt.Errorf("failed to commit namespace directory: %w", err)
	}

	return nil
}

// commitFilesToGitHub commits multiple files to GitHub in a single commit using Git API
func (s *GitOpsService) commitFilesToGitHub(ctx context.Context, ghClient *github.Client, owner, repo, branch string, files map[string]string, message string, config any) error {
	// Get the latest commit SHA for the branch
	ref, _, err := ghClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("failed to get branch ref: %w", err)
	}
	baseCommitSHA := ref.GetObject().GetSHA()

	// Get the base tree SHA
	baseCommit, _, err := ghClient.Git.GetCommit(ctx, owner, repo, baseCommitSHA)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}
	baseTreeSHA := baseCommit.GetTree().GetSHA()

	// Create blobs for each file
	var treeEntries []*github.TreeEntry
	for path, content := range files {
		blob, _, err := ghClient.Git.CreateBlob(ctx, owner, repo, &github.Blob{
			Content:  github.Ptr(content),
			Encoding: github.Ptr("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %w", path, err)
		}

		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(path),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  blob.SHA,
		})
	}

	// Create new tree
	tree, _, err := ghClient.Git.CreateTree(ctx, owner, repo, baseTreeSHA, treeEntries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Get commit author from config
	gitopsConfig, ok := config.(*models.GitOpsConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Create commit
	commit, _, err := ghClient.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message: github.Ptr(message),
		Tree:    tree,
		Parents: []*github.Commit{{SHA: github.Ptr(baseCommitSHA)}},
		Author: &github.CommitAuthor{
			Name:  github.Ptr(gitopsConfig.Username),
			Email: github.Ptr(gitopsConfig.Email),
		},
		Committer: &github.CommitAuthor{
			Name:  github.Ptr(gitopsConfig.Username),
			Email: github.Ptr(gitopsConfig.Email),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Update branch reference
	_, _, err = ghClient.Git.UpdateRef(ctx, owner, repo, &github.Reference{
		Ref: github.Ptr("refs/heads/" + branch),
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	}, false)
	if err != nil {
		return fmt.Errorf("failed to update ref: %w", err)
	}

	fmt.Printf("Successfully committed namespace manifests to %s/%s (branch: %s)\n", owner, repo, branch)
	return nil
}

// DeleteNamespaceManifests deletes the namespace directory from deployments/
func (s *GitOpsService) DeleteNamespaceManifests(ctx context.Context, namespaceName string) error {
	ghClient, err := s.GetGitHubClient()
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	owner, repo := ghClient.GetRepoInfo()
	branch := ghClient.GetRepoBranch()

	gitopsConfig, err := s.GetConfigOrDefault()
	if err != nil {
		return fmt.Errorf("failed to get GitOps config: %w", err)
	}

	// Delete the namespace directory from deployments/
	namespacePath := fmt.Sprintf("deployments/%s", namespaceName)
	commitMsg := fmt.Sprintf("Delete namespace %s", namespaceName)

	if err := s.deleteDirectoryFromGitHub(ctx, ghClient.Client, owner, repo, branch, namespacePath, commitMsg, gitopsConfig); err != nil {
		return fmt.Errorf("failed to delete namespace directory: %w", err)
	}

	return nil
}

// deleteDirectoryFromGitHub recursively deletes a directory from GitHub
func (s *GitOpsService) deleteDirectoryFromGitHub(ctx context.Context, ghClient *github.Client, owner, repo, branch, path, message string, config *models.GitOpsConfig) error {
	// Get the latest commit SHA for the branch
	ref, _, err := ghClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("failed to get branch ref: %w", err)
	}
	baseCommitSHA := ref.GetObject().GetSHA()

	// Get the base tree SHA
	baseCommit, _, err := ghClient.Git.GetCommit(ctx, owner, repo, baseCommitSHA)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}
	baseTreeSHA := baseCommit.GetTree().GetSHA()

	// Get the tree contents recursively
	tree, _, err := ghClient.Git.GetTree(ctx, owner, repo, baseTreeSHA, true)
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	// Filter out entries that start with the path to delete
	// Only include blob entries (not tree/directory entries)
	var treeEntries []*github.TreeEntry
	for _, entry := range tree.Entries {
		if entry.Path != nil && entry.Type != nil && *entry.Type == "blob" && !matchesPath(*entry.Path, path) {
			treeEntries = append(treeEntries, &github.TreeEntry{
				Path: entry.Path,
				Mode: entry.Mode,
				Type: entry.Type,
				SHA:  entry.SHA,
			})
		}
	}

	// Create new tree without the deleted directory (pass empty string)
	newTree, _, err := ghClient.Git.CreateTree(ctx, owner, repo, "", treeEntries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Create commit
	commit, _, err := ghClient.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message: github.Ptr(message),
		Tree:    newTree,
		Parents: []*github.Commit{{SHA: github.Ptr(baseCommitSHA)}},
		Author: &github.CommitAuthor{
			Name:  github.Ptr(config.Username),
			Email: github.Ptr(config.Email),
		},
		Committer: &github.CommitAuthor{
			Name:  github.Ptr(config.Username),
			Email: github.Ptr(config.Email),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Update branch reference
	_, _, err = ghClient.Git.UpdateRef(ctx, owner, repo, &github.Reference{
		Ref: github.Ptr("refs/heads/" + branch),
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	}, false)
	if err != nil {
		return fmt.Errorf("failed to update ref: %w", err)
	}

	fmt.Printf("Successfully deleted namespace directory %s from %s/%s (branch: %s)\n", path, owner, repo, branch)
	return nil
}

// matchesPath checks if a file path starts with the given directory path
func matchesPath(filePath, dirPath string) bool {
	if len(filePath) < len(dirPath) {
		return false
	}
	if filePath[:len(dirPath)] != dirPath {
		return false
	}
	// Exact match or starts with dirPath/
	return len(filePath) == len(dirPath) || (len(filePath) > len(dirPath) && filePath[len(dirPath)] == '/')
}

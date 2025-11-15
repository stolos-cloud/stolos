package gitops

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/config"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	githubpkg "github.com/stolos-cloud/stolos/backend/pkg/github"
	"gorm.io/gorm"
)

type GitOpsService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewGitOpsService(db *gorm.DB, cfg *config.Config) *GitOpsService {
	return &GitOpsService{
		db:  db,
		cfg: cfg,
	}
}

func (s *GitOpsService) IsConfiguredFromDatabase() bool {
	config, err := s.GetCurrentConfig()
	return err == nil && config != nil && config.IsConfigured
}

func (s *GitOpsService) IsConfiguredFromEnv() bool {
	return s.cfg.GitOps.RepoOwner != "" && s.cfg.GitOps.RepoName != ""
}

// initializes GitOps configuration on server startup
func (s *GitOpsService) InitializeGitOps(ctx context.Context) (*models.GitOpsConfig, error) {
	// Return existing config if already set up
	if s.IsConfiguredFromDatabase() {
		return s.GetCurrentConfig()
	}

	// If no DB config and no env config, skip initialization
	if !s.IsConfiguredFromEnv() {
		return nil, nil
	}

	// Initialize from env config
	return s.ConfigureGitOps(ctx, s.cfg.GitOps.RepoOwner, s.cfg.GitOps.RepoName, s.cfg.GitOps.Branch, s.cfg.GitOps.WorkingDir, s.cfg.GitOps.Username, s.cfg.GitOps.Email)
}

func (s *GitOpsService) ConfigureGitOps(ctx context.Context, repoOwner, repoName, branch, workingDir, username, email string) (*models.GitOpsConfig, error) {
	var dbConfig models.GitOpsConfig
	err := s.db.Where("is_configured = ?", true).First(&dbConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch existing config: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		dbConfig = models.GitOpsConfig{
			ID:           uuid.New(),
			IsConfigured: true,
		}
	}

	dbConfig.RepoOwner = repoOwner
	dbConfig.RepoName = repoName

	// Set branch with default
	if branch != "" {
		dbConfig.Branch = branch
	} else {
		dbConfig.Branch = "main"
	}

	// Set working dir with default
	if workingDir != "" {
		dbConfig.WorkingDir = workingDir
	} else {
		dbConfig.WorkingDir = "terraform"
	}

	// Set username with default
	if username != "" {
		dbConfig.Username = username
	} else {
		dbConfig.Username = "Stolos Bot"
	}

	// Set email with default
	if email != "" {
		dbConfig.Email = email
	} else {
		dbConfig.Email = "bot@stolos.cloud"
	}

	if err == gorm.ErrRecordNotFound {
		err = s.db.Create(&dbConfig).Error
	} else {
		err = s.db.Save(&dbConfig).Error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save GitOps config: %w", err)
	}

	return &dbConfig, nil
}

func (s *GitOpsService) GetCurrentConfig() (*models.GitOpsConfig, error) {
	var config models.GitOpsConfig
	err := s.db.Where("is_configured = ?", true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetConfigOrDefault returns the config from DB, or falls back to env/defaults
func (s *GitOpsService) GetConfigOrDefault() (*models.GitOpsConfig, error) {
	// Try DB first
	if s.IsConfiguredFromDatabase() {
		return s.GetCurrentConfig()
	}

	// Fall back to env/defaults
	if s.IsConfiguredFromEnv() {
		branch := s.cfg.GitOps.Branch
		if branch == "" {
			branch = "main"
		}

		workingDir := s.cfg.GitOps.WorkingDir
		if workingDir == "" {
			workingDir = "terraform"
		}

		username := s.cfg.GitOps.Username
		if username == "" {
			username = "Stolos Bot"
		}

		email := s.cfg.GitOps.Email
		if email == "" {
			email = "bot@stolos.cloud"
		}

		return &models.GitOpsConfig{
			RepoOwner:  s.cfg.GitOps.RepoOwner,
			RepoName:   s.cfg.GitOps.RepoName,
			Branch:     branch,
			WorkingDir: workingDir,
			Username:   username,
			Email:      email,
		}, nil
	}

	return nil, fmt.Errorf("GitOps not configured in database or environment")
}

// GetGitHubClient creates a GitHub client using app config + GitOps config
func (s *GitOpsService) GetGitHubClient() (*githubpkg.Client, error) {
	gitopsConfig, err := s.GetConfigOrDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps config: %w", err)
	}

	return githubpkg.NewClientFromConfig(
		s.cfg.GitHub.AppID,
		s.cfg.GitHub.InstallationID,
		s.cfg.GitHub.PrivateKey,
		gitopsConfig.RepoOwner,
		gitopsConfig.RepoName,
		gitopsConfig.Branch,
	)
}

func (s *GitOpsService) GetLatestCommitSHA() (string, error) {
	ctx := context.Background()
	ghClient, err := s.GetGitHubClient()
	config, err := s.GetConfigOrDefault()

	// Get the latest commit SHA for the branch (we refresh this before committing)
	ref, _, err := ghClient.Git.GetRef(ctx, config.RepoOwner, config.RepoName, "refs/heads/"+config.Branch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch ref: %w", err)
	}
	baseCommitSHA := ref.GetObject().GetSHA()

	// Get the base tree SHA
	baseCommit, _, err := ghClient.Git.GetCommit(ctx, config.RepoOwner, config.RepoName, baseCommitSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get base commit: %w", err)
	}
	return baseCommit.GetSHA(), nil
}

const (
	TemplateScaffold    string = "scaffolds"
	Templates           string = "templates"
	Namespaces          string = "namespaces"
	TemplateDeployments string = "apps"
)

func (s *GitOpsService) GetTemplateScaffolds() ([]string, error) {
	ctx := context.Background()
	ghClient, err := s.GetGitHubClient()
	if err != nil {
		return nil, fmt.Errorf("get github client: %w", err)
	}

	config, err := s.GetConfigOrDefault()
	if err != nil {
		return nil, fmt.Errorf("get gitops config: %w", err)
	}

	_, directoryContent, _, err := ghClient.Repositories.GetContents(ctx, config.RepoOwner, config.RepoName, TemplateScaffold, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch scaffold contents: %w", err)
	}

	var directories []string
	for _, content := range directoryContent {
		if content == nil {
			continue
		}

		if strings.EqualFold(content.GetType(), "dir") {
			directories = append(directories, content.GetName())
		}
	}

	return directories, nil
}

// GetDefaultBranchHeadRef returns the reference object for the default branch of a repo.
// Example ref.Ref = "refs/heads/main"
func GetDefaultBranchHeadRef(ctx context.Context, gh *github.Client, owner, repo string) (*github.Reference, error) {
	// 1. Get repo metadata (contains default branch name)
	repository, _, err := gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}
	defaultBranch := repository.GetDefaultBranch()
	if defaultBranch == "" {
		return nil, fmt.Errorf("repository has no default branch")
	}

	// 2. Get the ref for the default branch
	ref, _, err := gh.Git.GetRef(ctx, owner, repo, "heads/"+defaultBranch)
	if err != nil {
		return nil, fmt.Errorf("get ref for default branch %q: %w", defaultBranch, err)
	}

	return ref, nil
}

// DuplicateDirectory copies all blobs under srcPrefix -> dstPrefix by making a single commit.
// If overwrite is false, it aborts if any destination path already exists.
func (s *GitOpsService) DuplicateDirectory(srcPrefix, dstPrefix string, overwrite bool) error {
	ctx := context.Background()
	gitOpsConfig, _ := s.GetConfigOrDefault()
	owner := gitOpsConfig.RepoOwner
	repo := gitOpsConfig.RepoName
	gh, _ := s.GetGitHubClient()

	ref, _ := GetDefaultBranchHeadRef(ctx, gh.Client, owner, repo)
	headSHA, _ := s.GetLatestCommitSHA()

	// 1. Get recursive git tree
	tree, _, err := gh.Git.GetTree(ctx, owner, repo, headSHA, true /*recursive*/)
	if err != nil {
		return fmt.Errorf("get tree: %w", err)
	}

	// Normalize prefixes
	norm := func(s string) string { return strings.Trim(s, "/") }
	srcPrefix = norm(srcPrefix) + "/"
	dstPrefix = norm(dstPrefix) + "/"

	var entries []*github.TreeEntry
	existing := map[string]bool{} // destination path existence check

	// Build lookup of existing paths to detect collisions when overwrite=false
	for _, te := range tree.Entries {
		if te.Path != nil {
			existing[*te.Path] = true
		}
	}

	// 2. Create new TreeEntries in dst using the same blobs from src
	for _, te := range tree.Entries {
		if te.Path == nil || te.Type == nil || te.SHA == nil || te.Mode == nil {
			continue
		}
		if *te.Type != "blob" {
			// ignore subtrees/symlinks, etc
			// TODO : better handling here
			continue
		}
		if !strings.HasPrefix(*te.Path, srcPrefix) {
			continue
		}

		rel := strings.TrimPrefix(*te.Path, srcPrefix)
		dstPath := dstPrefix + rel

		if !overwrite && existing[dstPath] {
			return fmt.Errorf("destination already exists (set overwrite=true to replace): %s", dstPath)
		}

		entries = append(entries, &github.TreeEntry{
			Path: github.Ptr(dstPath),
			Mode: github.Ptr("100644"), // regular file
			Type: github.Ptr("blob"),
			SHA:  te.SHA, // reuse existing blob
		})
	}

	if len(entries) == 0 {
		return fmt.Errorf("no files found under %q", srcPrefix)
	}

	// 3. New tree with the new entries
	newTree, _, err := gh.Git.CreateTree(ctx, owner, repo, headSHA, entries)
	if err != nil {
		return fmt.Errorf("create tree: %w", err)
	}

	// 4. Commit
	msg := fmt.Sprintf("Copy %s -> %s", strings.TrimSuffix(srcPrefix, "/"), strings.TrimSuffix(dstPrefix, "/"))
	commit := &github.Commit{
		Message: github.String(msg),
		Tree:    newTree,
		Parents: []*github.Commit{{SHA: github.String(headSHA)}},
		Author: &github.CommitAuthor{
			Name:  github.Ptr(gitOpsConfig.Username),
			Email: github.Ptr(gitOpsConfig.Email),
		},
		Committer: &github.CommitAuthor{
			Name:  github.Ptr(gitOpsConfig.Username),
			Email: github.Ptr(gitOpsConfig.Email),
		},
	}

	newCommit, _, err := gh.Git.CreateCommit(ctx, owner, repo, commit, nil)
	if err != nil {
		return fmt.Errorf("create commit: %w", err)
	}

	// 5. Update HEAD to our commit // TODO : is this needed?
	ref.Object.SHA = newCommit.SHA
	if _, _, err := gh.Git.UpdateRef(ctx, owner, repo, ref, false /*force*/); err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	return nil
}

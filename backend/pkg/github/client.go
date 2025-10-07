package github

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v74/github"
)

type Config struct {
	AppID          int64  `json:"app_id"`
	PrivateKey     string `json:"private_key"`
	InstallationID int64  `json:"installation_id"`
	RepoOwner      string `json:"repo_owner"`
	RepoName       string `json:"repo_name"`
	Branch   string `json:"branch"`
}

type Client struct {
	*github.Client
	config *Config
}

func NewClientFromEnv() (*Client, error) {
	config, err := configFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	token, err := config.generateInstallationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate installation token: %w", err)
	}

	githubClient := github.NewClient(nil).WithAuthToken(token)

	return &Client{
		Client: githubClient,
		config: config,
	}, nil
}

func configFromEnv() (*Config, error) {
	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		return nil, fmt.Errorf("GITHUB_APP_ID environment variable is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	installationIDStr := os.Getenv("GITHUB_INSTALLATION_ID")
	if installationIDStr == "" {
		return nil, fmt.Errorf("GITHUB_INSTALLATION_ID environment variable is required")
	}
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_INSTALLATION_ID: %w", err)
	}

	privateKey := os.Getenv("GITHUB_PRIVATE_KEY")
	if privateKey == "" {
		return nil, fmt.Errorf("GITHUB_PRIVATE_KEY environment variable is required")
	}

	repoOwner := os.Getenv("GITOPS_REPO_OWNER")
	if repoOwner == "" {
		return nil, fmt.Errorf("GITOPS_REPO_OWNER environment variable is required")
	}

	repoName := os.Getenv("GITOPS_REPO_NAME")
	if repoName == "" {
		return nil, fmt.Errorf("GITOPS_REPO_NAME environment variable is required")
	}

	gitOpsBranch := os.Getenv("GITOPS_BRANCH")
	if gitOpsBranch == "" {
		gitOpsBranch = "main" // Default
	}

	return &Config{
		AppID:          appID,
		PrivateKey:     privateKey,
		InstallationID: installationID,
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		Branch:   gitOpsBranch,
	}, nil
}

func (c *Client) GetRepoInfo() (owner, name string) {
	return c.config.RepoOwner, c.config.RepoName
}

func (c *Client) GetConfig() *Config {
	return c.config
}

func (c *Client) GetRepoBranch() string {
	return c.config.Branch
}

//  creates a new installation access token using GitHub App credentials
func (c *Config) generateInstallationToken() (string, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	jwtToken, err := c.createJWTToken(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}

	installationToken, err := c.getInstallationAccessToken(jwtToken)
	if err != nil {
		return "", fmt.Errorf("failed to get installation access token: %w", err)
	}

	return installationToken, nil
}

func (c *Config) createJWTToken(privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": c.AppID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func (c *Config) getInstallationAccessToken(jwtToken string) (string, error) {
	client := github.NewClient(nil).WithAuthToken(jwtToken)

	installationToken, _, err := client.Apps.CreateInstallationToken(
		context.Background(),
		c.InstallationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return "", err
	}

	return installationToken.GetToken(), nil
}

func (c *Client) RefreshToken() error {
	token, err := c.config.generateInstallationToken()
	if err != nil {
		return fmt.Errorf("failed to generate new token: %w", err)
	}

	c.Client = github.NewClient(nil).WithAuthToken(token)
	return nil
}
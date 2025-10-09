package github

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func NewClientFromConfig(appID, installationID int64, privateKey, repoOwner, repoName, branch string) (*Client, error) {
	config := &Config{
		AppID:          appID,
		PrivateKey:     privateKey,
		InstallationID: installationID,
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		Branch:         branch,
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
	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprintf("%d", c.AppID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
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
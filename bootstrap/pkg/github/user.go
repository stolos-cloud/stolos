package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

type GitHubUser struct {
	Login string `json:"login"`
	Type  string `json:"type"` // "User" or "Organization"
}

// GetGitHubUserType queries the GitHub API to determine if a login is a User or Organization.
// token is optional.
func GetGitHubUserType(ctx context.Context, login, oauthToken oauth2.Token) (string, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", login)
	token := oauthToken.AccessToken

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var entity GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&entity); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// entity.Type will be "User" or "Organization"
	return entity.Type, nil
}

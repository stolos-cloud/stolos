package providers

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type GitHubProvider struct {
	clientID     string
	clientSecret string
}

func NewGitHubProvider(clientID, clientSecret string) *GitHubProvider {
	return &GitHubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (p *GitHubProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Scopes:       []string{"repo"},
		Endpoint:     endpoints.GitHub,
	}
}

func (p *GitHubProvider) GetName() string {
	return "GitHub"
}

func (p *GitHubProvider) GetCallbackPath() string {
	return "/oauth/github/callback"
}
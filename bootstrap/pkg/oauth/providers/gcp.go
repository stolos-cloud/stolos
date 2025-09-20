package providers

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GCPProvider struct {
	clientID     string
	clientSecret string
}

func NewGCPProvider(clientID, clientSecret string) *GCPProvider {
	return &GCPProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (p *GCPProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/iam",
		},
		Endpoint: google.Endpoint,
	}
}

func (p *GCPProvider) GetName() string {
	return "GCP"
}

func (p *GCPProvider) GetCallbackPath() string {
	return "/oauth/gcp/callback"
}
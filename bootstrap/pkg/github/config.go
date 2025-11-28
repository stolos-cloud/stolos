package github

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/k8s"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/logger"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/oauth"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/oauth/providers"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_json "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
)

// set via ldflags
var GithubOauthClientId string
var GithubOauthClientSecret string

type OauthClient struct {
	*github.Client
	token *oauth2.Token
}

// legacy method
func NewOauthClient(token *oauth2.Token) *OauthClient {
	client := github.NewClient(nil).WithAuthToken(token.AccessToken)
	return &OauthClient{
		Client: client,
		token:  token,
	}
}

// response from GitHub when generating an installation access token
type GitHubAppAccessToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Create installation access token given GitHub App installation
func GenerateGitHubAccessToken(ctx context.Context, appID int64, privateKeyPEM string, installationID string) (string, error) {
	// Generate JWT for app authentication
	jwtToken, err := generateAppJWT(appID, privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Request installation access token
	apiURL := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status %d: body %s", resp.StatusCode, string(body))
	}

	var tokenResponse GitHubAppAccessToken
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	return tokenResponse.Token, nil
}

// creates a JWT for authenticating as a GitHub App
func generateAppJWT(appID int64, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprintf("%d", appID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)), // must be <= 10m
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(key)
}

// NewClientFromApp creates a GitHub client using GitHub App credentials
func NewClientFromApp(ctx context.Context, appID int64, privateKeyPEM string, installationID string) (*OauthClient, error) {
	installationToken, err := GenerateGitHubAccessToken(ctx, appID, privateKeyPEM, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate installation token: %w", err)
	}

	client := github.NewClient(nil).WithAuthToken(installationToken)
	return &OauthClient{
		Client: client,
		token:  nil, // No OAuth token for app-based auth
	}, nil
}

// GitHubInfo contains repository setup information
type GitHubInfo struct {
	RepoOwner      string `json:"RepoOwner" field_label:"Github Organization Name" field_required:"true"`
	RepoName       string `json:"RepoName" field_label:"Github Repository Name" field_required:"true"`
	BaseDomain     string `json:"BaseDomain" field_label:"Base Domain (DNS)" field_required:"true"`
	LoadBalancerIP string `json:"LoadBalancerIP" field_label:"LoadBalancer IP" field_required:"true"`
	PackagesPAT    string `json:"PackagesPAT" field_label:"GitHub PAT for Packages (read:packages)" field_required:"true"`
}

// Config contains GitHub credentials for backend usage
type Config struct {
	RepoOwner      string `json:"repo_owner"`
	RepoName       string `json:"repo_name"`
	AppID          string `json:"app_id,omitempty"`
	AppPEM         string `json:"app_pem,omitempty"`
	InstallationID string `json:"installation_id,omitempty"`
}

func (client *OauthClient) InitRepo(info *GitHubInfo, isPrivate bool) (*github.Repository, error) {
	templateRepoOwner := os.Getenv("GITHUB_TEMPLATE_REPO_OWNER")
	templateRepoName := os.Getenv("GITHUB_TEMPLATE_REPO_NAME")
	if templateRepoOwner == "" {
		templateRepoOwner = "stolos-cloud"
	}
	if templateRepoName == "" {
		templateRepoName = "stolos-k8s-template"
	}

	repo, response, err := client.Repositories.CreateFromTemplate(context.Background(), templateRepoOwner, templateRepoName, &github.TemplateRepoRequest{
		Name:               &info.RepoName,
		Owner:              &info.RepoOwner,
		IncludeAllBranches: github.Ptr(false),
		Private:            &isPrivate,
	})

	if err != nil {
		return nil, err
	}

	if response.StatusCode != 201 {
		return nil, fmt.Errorf("CreateFromTemplate returned %d", response.StatusCode)
	}

	time.Sleep(5 * time.Second) // Wait for github to init repo, as createfile can happen before it is fully initialized

	// Create initial config file
	//if err := client.createInitialConfig(info); err != nil {
	//	return nil, fmt.Errorf("failed to create initial config: %w", err)
	//}

	return repo, nil
}

// createInitialConfig creates the initial common.yml configuration file
func (c *OauthClient) CreateInitialConfig(config *unstructured.Unstructured, info *GitHubInfo) error {

	s := k8s_json.NewSerializerWithOptions(
		k8s_json.DefaultMetaFactory,
		nil, // scheme — nil works for unstructured
		nil,
		k8s_json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: false,
		},
	)

	var out bytes.Buffer
	err := s.Encode(config, &out)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	author := github.CommitAuthor{
		Name:  github.Ptr("Bot Stolos"),
		Email: github.Ptr("bot@stolos.cloud"),
		Date: &github.Timestamp{
			Time: time.Now(),
		},
	}

	filePath := "system/stolos-system.yaml"

	// Check if file already exists to get its SHA for update
	existingFile, _, _, _ := c.Repositories.GetContents(
		context.Background(),
		info.RepoOwner,
		info.RepoName,
		filePath,
		&github.RepositoryContentGetOptions{Ref: "main"},
	)

	opts := &github.RepositoryContentFileOptions{
		Message:   github.Ptr("Update stolos config file"),
		Content:   out.Bytes(),
		Branch:    github.Ptr("main"),
		Committer: &author,
	}

	// If file exists, include SHA to update it
	if existingFile != nil {
		opts.SHA = existingFile.SHA
	} else {
		opts.Message = github.Ptr("Initial config file")
	}

	_, response, err := c.Repositories.CreateFile(
		context.Background(),
		info.RepoOwner,
		info.RepoName,
		filePath,
		opts,
	)

	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	if response.StatusCode != 200 && response.StatusCode != 201 {
		return fmt.Errorf("CreateFile returned %d", response.StatusCode)
	}

	return nil
}

// GetToken returns the OAuth token
func (c *OauthClient) GetToken() *oauth2.Token {
	return c.token
}

// AuthenticateAndSetup performs OAuth authentication and repository initialization
func AuthenticateAndSetup(oauthServer *oauth.Server, clientID, clientSecret string, info *GitHubInfo, logger logger.Logger) (*OauthClient, error) {
	ctx := context.Background()

	provider := providers.NewGitHubProvider(clientID, clientSecret)
	oauthServer.RegisterProvider(provider)

	token, err := oauthServer.Authenticate(ctx, "GitHub")
	if err != nil {
		return nil, fmt.Errorf("GitHub authentication failed: %w", err)
	}

	client := NewOauthClient(token)

	_, err = client.InitRepo(info, false)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	logger.Infof("Repo initialized: https://github.com/%s/%s.git", info.RepoOwner, info.RepoName)
	return client, nil
}

// creates a new GitHub configuration
func NewGithubAppConfig(repoOwner, repoName, appID, appPEM, installationID string) *Config {
	return &Config{
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		AppID:          appID,
		AppPEM:         appPEM,
		InstallationID: installationID,
	}
}

// ToSecret serializes GitHub config to Kubernetes secret
func (c *Config) ToSecret(namespace, secretName string) *corev1.Secret {
	data := map[string][]byte{
		"GITHUB_REPO_OWNER":      []byte(c.RepoOwner),
		"GITHUB_REPO_NAME":       []byte(c.RepoName),
		"GITHUB_APP_ID":          []byte(c.AppID),
		"GITHUB_APP_PRIVATE_KEY": []byte(c.AppPEM),
		"GITHUB_INSTALLATION_ID": []byte(c.InstallationID),
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "stolos-platform",
				"app.kubernetes.io/component": "backend",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

// FromSecret deserializes Kubernetes secret to GitHub config
func FromSecret(secret *corev1.Secret) (*Config, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret data is nil")
	}

	return &Config{
		RepoOwner:      string(secret.Data["GITHUB_REPO_OWNER"]),
		RepoName:       string(secret.Data["GITHUB_REPO_NAME"]),
		AppID:          string(secret.Data["GITHUB_APP_ID"]),
		AppPEM:         string(secret.Data["GITHUB_APP_PRIVATE_KEY"]),
		InstallationID: string(secret.Data["GITHUB_INSTALLATION_ID"]),
	}, nil
}

// CreateOrUpdateSecret creates or updates the GitHub config secret in Kubernetes
func (c *Config) CreateOrUpdateSecret(ctx context.Context, client kubernetes.Interface, namespace, secretName string) error {
	secret := c.ToSecret(namespace, secretName)
	return k8s.CreateOrUpdateSecret(ctx, client, secret, true)
}

// CreateOrUpdateArgoCDGitHubSecrets ensures both ArgoCD repository and notifications secrets exist and are up-to-date.
func CreateOrUpdateArgoCDGitHubSecrets(
	ctx context.Context,
	client kubernetes.Interface,
	namespace, secretName string,
	appID string,
	appPEM string,
	repoUrl string,
	installID string,
) error {
	// TODO : Does not support GitHub Enterprise.

	// 1. ArgoCD GH Repo secret
	repoSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"type":                    "git",
			"url":                     repoUrl,
			"githubAppID":             appID,
			"githubAppPrivateKey":     appPEM,
			"githubAppInstallationID": installID,
		},
	}

	if err := k8s.CreateOrUpdateSecret(ctx, client, repoSecret, false); err != nil {
		return fmt.Errorf("failed to apply ArgoCD repo secret: %w", err)
	}

	// TODO : Add Notifications controller secret
	// 2. ArgoCD Notifications GitHub service
	//tokenSecret := &corev1.Secret{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      fmt.Sprintf("%s-notifications", secretName),
	//		Namespace: namespace,
	//		Labels: map[string]string{
	//			"argocd-notifications.argoproj.io/secret-type": "github",
	//		},
	//	},
	//	Type: corev1.SecretTypeOpaque,
	//	StringData: map[string]string{
	//		"github-privateKey": app.PEM,
	//	},
	//}

	//TODO: Add Notifications CM config (Notifications):
	// 			"appID":             fmt.Sprintf("%d", app.ID),
	//			"installationID":    fmt.Sprintf("%d", install.ID),
	//if err := createOrUpdateSecret(ctx, client, tokenSecret); err != nil {
	//	return fmt.Errorf("failed to apply ArgoCD notifications secret: %w", err)
	//}

	return nil
}

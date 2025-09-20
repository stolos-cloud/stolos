package github

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v74/github"
	"github.com/pkg/browser"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

var GithubClientId string     // To set using ldflags
var GithubClientSecret string // To set using ldflags

func AuthenticateGithubClient(log *tui.UILogger) (*github.Client, error) {
	config := &oauth2.Config{
		ClientID:     GithubClientId,
		ClientSecret: GithubClientSecret,
		Scopes:       []string{"repo"},
		Endpoint:     endpoints.GitHub,
		RedirectURL:  "http://localhost:9999/oauth/callback",
	}

	/*---Code emprunté: https://www.iamyadav.com/blogs/how-to-authenticate-cli-using-oauth ---
	  Des modifications y sont apportés afin de le rendre compatible avec notre structure */

	// start server
	ctx := context.Background()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	sslcli := &http.Client{Transport: tr}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, sslcli)

	server := &http.Server{Addr: ":9999"}

	// create a channel to receive the authorization code
	codeChan := make(chan string)

	http.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		queryParts, _ := url.ParseQuery(r.URL.RawQuery)

		// Use the authorization code that is pushed to the redirect URL.
		code := queryParts["code"][0]

		// write the authorization code to the channel
		codeChan <- code

		msg := "<p><strong>Authentication successful</strong>. You may now close this tab.</p>"
		// send a success message to the browser
		fmt.Fprint(w, msg)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// get the OAuth authorization URL
	oauthUrl := config.AuthCodeURL("state", oauth2.AccessTypeOffline)

	// Redirect user to consent page to ask for permission
	// for the scopes specified above
	log.Infof("Your browser has been opened to visit::\n%s\n", oauthUrl)

	// open user's browser to login page
	if err := browser.OpenURL(oauthUrl); err != nil {
		log.Errorf("failed to open browser for authentication %s", err.Error())
		log.Infof("Copy-paste link in your browser to continue: %s", oauthUrl)
	}

	// wait for the authorization code to be received
	code := <-codeChan

	// exchange the authorization code for an access token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Errorf("Failed to exchange authorization code for token: %v", err)
		return nil, err
	}

	if !token.Valid() {
		return nil, errors.New("Can't get source information without accessToken")
	}

	// shut down the HTTP server
	if err := server.Shutdown(context.Background()); err != nil {
		log.Errorf("Failed to shut down server: %v", err)
	}

	log.Success("Authentication successful")

	client := github.NewClient(nil).WithAuthToken(token.AccessToken)

	return client, nil

	/*--- Fin du code emprunté --- */
}

func InitRepo(client *github.Client, info *state.BootstrapInfo, isPrivate bool) (*github.Repository, error) {
	templateRepoOwner := os.Getenv("GITHUB_TEMPLATE_REPO_OWNER")
	templateRepoName := os.Getenv("GITHUB_TEMPLATE_REPO_NAME")
	if templateRepoOwner == "" {
		templateRepoOwner = "Simon-Boyer"
	}
	if templateRepoName == "" {
		templateRepoName = "etsmtl-pfe-cloudnative-template"
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

	commonConfig := struct {
		BaseDomain string `yaml:"base_domain"`
		LbIp       string `yaml:"lb_ip"`
	}{
		BaseDomain: info.BaseDomain,
		LbIp:       info.LoadBalancerIp,
	}
	commonConfigYaml, err := yaml.Marshal(commonConfig)
	author := github.CommitAuthor{
		Name:  github.Ptr("Bot Stolos"),
		Email: github.Ptr("bot@stolos.cloud"),
		Date: &github.Timestamp{
			Time: time.Now(),
		},
	}

	_, response, err = client.Repositories.CreateFile(context.Background(), info.RepoOwner, info.RepoName, "common.yml", &github.RepositoryContentFileOptions{
		Message:   github.Ptr("Initial config file"),
		Content:   commonConfigYaml,
		Branch:    github.Ptr("main"),
		Committer: &author,
	})

	if err != nil {
		return nil, err
	}

	if response.StatusCode != 201 {
		return nil, fmt.Errorf("CreateFromTemplate returned %d", response.StatusCode)
	}

	return repo, nil
}

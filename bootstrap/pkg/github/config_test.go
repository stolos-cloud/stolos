package github

import (
	"context"
	"os"
	"testing"

	"github.com/stolos-cloud/stolos-bootstrap/pkg/k8s"
	"k8s.io/client-go/kubernetes"
)

//func TestAuthenticateAndSetup(t *testing.T) {
//	type args struct {
//		oauthServer  *oauth.Server
//		clientID     string
//		clientSecret stringW
//		info         *GitHubInfo
//		logger       logger.Logger
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *Client
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := AuthenticateAndSetup(tt.args.oauthServer, tt.args.clientID, tt.args.clientSecret, tt.args.info, tt.args.logger)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("AuthenticateAndSetup() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("AuthenticateAndSetup() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestClient_GetToken(t *testing.T) {
//	type fields struct {
//		Client *github.Client
//		token  *oauth2.Token
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *oauth2.Token
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Client{
//				Client: tt.fields.Client,
//				token:  tt.fields.token,
//			}
//			if got := c.GetToken(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetToken() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestClient_InitRepo(t *testing.T) {
//	type fields struct {
//		Client *github.Client
//		token  *oauth2.Token
//	}
//	type args struct {
//		info      *GitHubInfo
//		isPrivate bool
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *github.Repository
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			client := &Client{
//				Client: tt.fields.Client,
//				token:  tt.fields.token,
//			}
//			got, err := client.InitRepo(tt.args.info, tt.args.isPrivate)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("InitRepo() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("InitRepo() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestClient_createInitialConfig(t *testing.T) {
//	type fields struct {
//		Client *github.Client
//		token  *oauth2.Token
//	}
//	type args struct {
//		info *GitHubInfo
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Client{
//				Client: tt.fields.Client,
//				token:  tt.fields.token,
//			}
//			if err := c.createInitialConfig(tt.args.info); (err != nil) != tt.wantErr {
//				t.Errorf("createInitialConfig() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestConfig_CreateOrUpdateSecret(t *testing.T) {
//	type fields struct {
//		AccessToken string
//		RepoOwner   string
//		RepoName    string
//	}
//	type args struct {
//		ctx        context.Context
//		client     kubernetes.Interface
//		namespace  string
//		secretName string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Config{
//				AccessToken: tt.fields.AccessToken,
//				RepoOwner:   tt.fields.RepoOwner,
//				RepoName:    tt.fields.RepoName,
//			}
//			if err := c.CreateOrUpdateSecret(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.secretName); (err != nil) != tt.wantErr {
//				t.Errorf("CreateOrUpdateSecret() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestConfig_ToSecret(t *testing.T) {
//	type fields struct {
//		AccessToken string
//		RepoOwner   string
//		RepoName    string
//	}
//	type args struct {
//		namespace  string
//		secretName string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *corev1.Secret
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Config{
//				AccessToken: tt.fields.AccessToken,
//				RepoOwner:   tt.fields.RepoOwner,
//				RepoName:    tt.fields.RepoName,
//			}
//			if got := c.ToSecret(tt.args.namespace, tt.args.secretName); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("ToSecret() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

//	func TestNewClient(t *testing.T) {
//		type args struct {
//			token *oauth2.Token
//		}
//		tests := []struct {
//			name string
//			args args
//			want *Client
//		}{
//			// TODO: Add test cases.
//		}
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				if got := NewClient(tt.args.token); !reflect.DeepEqual(got, tt.want) {
//					t.Errorf("NewClient() = %v, want %v", got, tt.want)
//				}
//			})
//		}
//	}
//
//	func TestNewConfig(t *testing.T) {
//		type args struct {
//			token     *oauth2.Token
//			repoOwner string
//			repoName  string
//		}
//		tests := []struct {
//			name string
//			args args
//			want *Config
//		}{
//			// TODO: Add test cases.
//		}
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				if got := NewConfig(tt.args.token, tt.args.repoOwner, tt.args.repoName); !reflect.DeepEqual(got, tt.want) {
//					t.Errorf("NewConfig() = %v, want %v", got, tt.want)
//				}
//			})
//		}
//	}
//func TestFromSecret(t *testing.T) {
//	type args struct {
//		secret *corev1.Secret
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *Config
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := FromSecret(tt.args.secret)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("FromSecret() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("FromSecret() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func getClient() kubernetes.Interface {
	kubeconfig, _ := os.ReadFile("./kubeconfig")
	k8sClient, _ := k8s.NewClientFromKubeconfig(kubeconfig)
	return k8sClient
}

func TestCreateOrUpdateArgoCDGitHubSecrets(t *testing.T) {
	type args struct {
		ctx        context.Context
		client     kubernetes.Interface
		namespace  string
		secretName string
		app        *AppManifest
		repoUrl    string
		install    *AppInstallation
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "TestArgoSecrets", args: struct {
			ctx        context.Context
			client     kubernetes.Interface
			namespace  string
			secretName string
			app        *AppManifest
			repoUrl    string
			install    *AppInstallation
		}{ctx: context.Background(), client: getClient(), namespace: "stolos-argocd", secretName: "stolos-github-app", app: &AppManifest{
			ID:            0,
			ClientID:      "d",
			ClientSecret:  "h",
			WebhookSecret: "i",
			PEM:           "PEM",

			Owner: User{
				Login:   "stolos-cloud",
				htmlURL: "",
				Type:    "",
			},
		}, repoUrl: "stolos-test-1", install: &AppInstallation{
			ID:                  123456,
			RepositorySelection: "asdfg",
		}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateOrUpdateArgoCDGitHubSecrets(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.secretName, tt.args.app, tt.args.repoUrl, tt.args.install); (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateArgoCDGitHubSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//func Test_createOrUpdateSecret(t *testing.T) {
//	type args struct {
//		ctx    context.Context
//		client kubernetes.Interface
//		secret *corev1.Secret
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := createOrUpdateSecret(tt.args.ctx, tt.args.client, tt.args.secret); (err != nil) != tt.wantErr {
//				t.Errorf("createOrUpdateSecret() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

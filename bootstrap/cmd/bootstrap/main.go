// main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/stolos-cloud/stolos-bootstrap/internal/configserver"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/gcp"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/github"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/helm"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/k8s"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/marshal"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/oauth"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"golang.org/x/oauth2"

	"github.com/cavaliergopher/grab/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siderolabs/image-factory/pkg/schematic"
)

var bootstrapInfos = &state.BootstrapInfo{}
var doRestoreProgress = false
var didReadBootstrapInfos = false

// var tui.Steps []tui.Step
var kubeconfig []byte
var saveState state.SaveState
var gcpConfig *gcp.Config
var githubConfig *github.Config
var oauthServer *oauth.Server
var githubToken *oauth2.Token
var gcpToken *oauth2.Token
var gcpEnabled = gcp.GCPClientId != "" && gcp.GCPClientSecret != ""
var gitHubEnabled = github.GithubClientId != "" && github.GithubClientSecret != ""

func main() {

	tui.RegisterDefaultFunc("GetOutboundIP", GetOutboundIP)

	_, err := os.Stat("bootstrap-state.json")

	if !(errors.Is(err, os.ErrNotExist)) {
		saveState = marshal.ReadStateFromJSON()
		bootstrapInfos = &saveState.BootstrapInfo
		didReadBootstrapInfos = true
		doRestoreProgress = true
	}

	_, err = os.Stat("bootstrap-config.json")
	if !(errors.Is(err, os.ErrNotExist)) {
		marshal.ReadBootstrapInfos("bootstrap-config.json", bootstrapInfos)
		didReadBootstrapInfos = true
	}

	if !didReadBootstrapInfos {
		bootstrapInfos = &state.BootstrapInfo{}
	}

	if didReadBootstrapInfos {
		// TODO
		//talosStep.AutoAdvance = true
		//githubStep.AutoAdvance = true
	}

	githubInfoStep := tui.Step{
		Name:        "GitHubInfo",
		Title:       "1) Enter GitHub Repository Information",
		Kind:        tui.StepForm,
		Fields:      tui.CreateFieldsForStruct[github.GitHubInfo](),
		IsDone:      true,
		AutoAdvance: false,
		OnExit: func(m *tui.Model, s *tui.Step) {
			if !didReadBootstrapInfos {
				githubInfo, err := tui.RetrieveStructFromFields[github.GitHubInfo](s.Fields)
				if err != nil {
				}
				bootstrapInfos.GitHubInfo = *githubInfo
			}
		},
	}

	githubAuthStep := tui.Step{
		Name:        "GitHubAuth",
		Title:       "1.1) Authenticate with GitHub",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunGitHubAuthStepInBackground,
		OnExit: func(m *tui.Model, s *tui.Step) {
			// TODO CHECK GH AUTH
		},
	}

	githubRepoStep := tui.Step{
		Name:        "GitHubRepo",
		Title:       "1.2) Create GitHub Repository",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunGitHubRepoStepInBackground,
		//OnExit: //TODO validate,
	}

	githubAppStep := tui.Step{
		// TODO: Implement GitHub App Manifest Flow
		Name:        "GitHubApp",
		Title:       "1.3) Create GitHub App",
		Kind:        tui.StepSpinner,
		IsDone:      true,
		AutoAdvance: true,
	}

	gcpInfoStep := tui.Step{
		Name:        "GCPInfo",
		Title:       "2) GCP Information",
		Kind:        tui.StepForm,
		Fields:      tui.CreateFieldsForStruct[state.GCPInfo](),
		IsDone:      true,
		AutoAdvance: false,
		OnEnter: func(m *tui.Model, s *tui.Step) tea.Cmd {
			// TODO
			return nil
		},
		OnExit: func(m *tui.Model, s *tui.Step) {
			if !didReadBootstrapInfos {
				gcpInfo, err := tui.RetrieveStructFromFields[state.GCPInfo](s.Fields)
				if err != nil {
				}
				bootstrapInfos.GCPInfo = *gcpInfo
			}
		},
	}

	gcpAuthStep := tui.Step{
		Name:        "GCPAuth",
		Title:       "2.1) GCP Authentication",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter: func(m *tui.Model, s *tui.Step) tea.Cmd {
			if !gcpEnabled {
				return nil
			}
			go func() {
				gcpToken, err = oauthServer.Authenticate(context.Background(), "GCP")
				if err != nil {
					m.Logger.Errorf("Failed to authenticate with GCP: %v", err)
					s.IsDone = true
					// TODO handle fail
				}
				s.IsDone = true
			}()
			return nil
		},
	}

	gcpSAStep := tui.Step{
		Name:        "GCPServiceAccount",
		Title:       "2.2) GCP Service Account",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunGCPSAStepInBackground,
	}

	talosInfoStep := tui.Step{
		Name:        "TalosInfo",
		Title:       "3) Talos and Kubernetes Information",
		Kind:        tui.StepForm,
		Fields:      tui.CreateFieldsForStruct[state.TalosInfo](),
		IsDone:      true,
		AutoAdvance: false,
		OnExit: func(m *tui.Model, s *tui.Step) {
			if !didReadBootstrapInfos {
				talosInfo, err := tui.RetrieveStructFromFields[state.TalosInfo](s.Fields)
				if err != nil {
				}
				bootstrapInfos.TalosInfo = *talosInfo
			}
		},
	}

	talosISOStep := tui.Step{
		Name:        "TalosISOStep",
		Title:       "3.1) Download Talos ISO",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunTalosISOStep,
	}

	waitControlPlaneStep := tui.Step{
		Name:        "WaitControlPlaneStep",
		Title:       "3.2) Wait Control Plane",
		Kind:        tui.StepSpinner,
		IsDone:      false, // Set by server
		AutoAdvance: true,
		OnEnter:     RunStartMachineconfigServerStep,
	}

	waitWorkerStep := tui.Step{
		Name:        "WaitWorkerStep",
		Title:       "3.3) Wait Worker",
		Kind:        tui.StepSpinner,
		IsDone:      true,  // Set by server
		AutoAdvance: false, // TODO bypassed for now via ENTER key
	}

	clusterBootstrapStep := tui.Step{
		Name:        "ClusterBootstrap",
		Title:       "3.4) Bootstrap Kubernetes",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunClusterBootstrapStepInBackground,
	}

	deployArgoStep := tui.Step{
		Name:        "DeployArgo",
		Title:       "4.1) Deploy Argo",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunArgoStepInBackground,
	}

	deployPortalStep := tui.Step{
		Name:        "DeployPortal",
		Title:       "4.2) Deploy Portal",
		Kind:        tui.StepSpinner,
		IsDone:      false,
		AutoAdvance: true,
		OnEnter:     RunPortalStepInBackground,
	}

	// Skip all gcp steps if not enabled
	tui.DisableStep(&gcpInfoStep, !gcpEnabled)
	tui.DisableStep(&gcpAuthStep, !gcpEnabled)
	tui.DisableStep(&gcpSAStep, !gcpEnabled)

	// Skip all github steps if not enabled
	tui.DisableStep(&githubInfoStep, !gitHubEnabled)
	tui.DisableStep(&githubAuthStep, !gitHubEnabled)
	tui.DisableStep(&githubRepoStep, !gitHubEnabled)
	tui.DisableStep(&githubAppStep, !gitHubEnabled)

	// Attn. sa fait des copies ici !
	tui.Steps = []*tui.Step{
		&githubInfoStep,
		&githubAuthStep,
		&githubRepoStep,
		&githubAppStep,
		&gcpInfoStep,
		&gcpAuthStep,
		&gcpSAStep,
		&talosInfoStep,
		&talosISOStep,
		&waitControlPlaneStep,
		&waitWorkerStep,
		&clusterBootstrapStep,
		&deployArgoStep,
		&deployPortalStep,
	}

	p, model := tui.NewWizard(tui.Steps)

	// Setup the OAuth providers and get feature enablement state
	oauthServer = SetupOAuthServer(model.Logger)
	if gcpEnabled {
		SetupGCP()
	}
	if gitHubEnabled {
		SetupGitHub()
	}

	if gitHubEnabled || gcpEnabled {
		RunOAuthServerInBackround(model.Logger)
	}

	// Run will block.
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}

}

func RunOAuthServerInBackround(logger *tui.UILogger) {
	go func() {
		ctx := context.Background()
		defer oauthServer.Stop(ctx)
		if err := oauthServer.Start(ctx); err != nil {
			logger.Errorf("skipping, oauth server start failed: %v", err)
			tui.Steps[1].IsDone = true
			return
		}
		// Keep the server running
		select {}
	}()
}

func SetupOAuthServer(logger *tui.UILogger) *oauth.Server {
	// Setup OAuth server
	server := oauth.NewServer("9999", logger)
	return server
}

func SetupGitHub() {
	if github.GithubClientId != "" && github.GithubClientSecret != "" {
		githubProvider := oauth.NewGitHubProvider(github.GithubClientId, github.GithubClientSecret)
		oauthServer.RegisterProvider(githubProvider)
	}
}

func SetupGCP() {
	if gcp.GCPClientId != "" && gcp.GCPClientSecret != "" {
		gcpProvider := oauth.NewGCPProvider(gcp.GCPClientId, gcp.GCPClientSecret)
		oauthServer.RegisterProvider(gcpProvider)
	}
}

func RunGCPSAStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {

	if !gcpEnabled {
		return nil
	}

	m.Logger.Infof("Read bootstrap infos from file, clusterName: %s", bootstrapInfos.TalosInfo.ClusterName)

	go func() {
		if gcp.GCPClientId != "" && gcp.GCPClientSecret != "" {
			// Create GCP service account
			_, err := gcp.CreateServiceAccountWithOAuth(
				context.Background(),
				bootstrapInfos.GCPInfo.GCPProjectID,
				bootstrapInfos.GCPInfo.GCPRegion,
				gcpToken,
				"stolos-platform-sa",
			)
			if err != nil {
				m.Logger.Errorf("Failed to create GCP service account: %v", err)
			} else {
				m.Logger.Success("GCP service account created successfully")
			}
		} else {
			m.Logger.Infof("GCP OAuth credentials not provided, skipping GCP service account creation")
		}

		s.IsDone = true
	}()
	return nil
}

func RunGitHubRepoStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	m.Logger.Infof("Creating github repo %s...", bootstrapInfos.GitHubInfo.RepoName)
	// Create GitHub client and initialize repository
	githubClient := github.NewClient(githubToken)
	githubBootstrapInfo := &github.GitHubInfo{
		RepoName:       bootstrapInfos.GitHubInfo.RepoName,
		RepoOwner:      bootstrapInfos.GitHubInfo.RepoOwner,
		BaseDomain:     bootstrapInfos.GitHubInfo.BaseDomain,
		LoadBalancerIP: bootstrapInfos.GitHubInfo.LoadBalancerIP,
	}

	_, err := githubClient.InitRepo(githubBootstrapInfo, false)
	if err != nil {
		m.Logger.Errorf("github init repo failed: %v", err)
	}

	// Create GitHub config for backend
	githubConfig = github.NewConfig(githubToken, bootstrapInfos.GitHubInfo.RepoOwner, bootstrapInfos.GitHubInfo.RepoName)

	m.Logger.Successf("Repo initialized! : https://github.com/%s/%s.git", bootstrapInfos.GitHubInfo.RepoOwner, bootstrapInfos.GitHubInfo.RepoName)
	s.IsDone = true
	return nil
}

func RunGitHubAuthStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	go func() {
		var err error
		githubToken, err = oauthServer.Authenticate(context.Background(), "GitHub")
		if err != nil {
			m.Logger.Errorf("skipping, oauth server authenticate failed: %v", err)
			s.IsDone = true
			return
		}
		s.IsDone = true
	}()
	return nil
}

func RunStartMachineconfigServerStep(m *tui.Model, s *tui.Step) tea.Cmd {
	m.Logger.Infof("Cluster: %s", bootstrapInfos.TalosInfo.ClusterName)
	addr := bootstrapInfos.TalosInfo.HTTPHostname + ":" + bootstrapInfos.TalosInfo.HTTPPort
	StartMachineconfigServerInBackground(m, addr)
	s.IsDone = true
	return nil
}

func RunTalosISOStep(m *tui.Model, s *tui.Step) tea.Cmd {
	if doRestoreProgress {
		m.Logger.Info("State file found, skipping to tui.Steps[5]")
		addr := bootstrapInfos.TalosInfo.HTTPHostname + ":" + bootstrapInfos.TalosInfo.HTTPPort
		StartMachineconfigServerInBackground(m, addr)
		m.CurrentStepIndex = 5
		return nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		talosConfigArg := fmt.Sprintf("talos.config=http://%s:%s/machineconfig?m=${mac}&u=${uuid}", bootstrapInfos.TalosInfo.HTTPHostname, bootstrapInfos.TalosInfo.HTTPPort)
		kernelArgs := []string{talosConfigArg, bootstrapInfos.TalosInfo.TalosExtraArgs}

		m.Logger.Infof("Generating image with kernelParam: %s", talosConfigArg)

		factory := talos.CreateFactoryClient()
		sch := schematic.Schematic{
			Overlay: schematic.Overlay{
				Image: bootstrapInfos.TalosInfo.TalosOverlayImage,
				Name:  bootstrapInfos.TalosInfo.TalosOverlayName,
				// Options: nil, // ==> Extra YAML settings passed to overlay image.
			},
			Customization: schematic.Customization{
				ExtraKernelArgs: kernelArgs,
			},
		}

		schematicId, _ := factory.SchematicCreate(ctx, sch)
		m.Logger.Infof("Generated schematicId: %s", schematicId)

		talosImageFormat := "iso"
		if bootstrapInfos.TalosInfo.TalosArchitecture == "arm64" {
			talosImageFormat = "raw.xz"
		}

		talosImagePath := fmt.Sprintf("metal-%s.%s", bootstrapInfos.TalosInfo.TalosArchitecture, talosImageFormat)
		talosImageUrl := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/%s", schematicId, bootstrapInfos.TalosInfo.TalosVersion, talosImagePath)
		//m.Logger.Infof("%s", talosImageUrl)

		m.Logger.Infof("Downloading image from %s...", talosImageUrl)
		resp, err := grab.Get(".", talosImageUrl)
		if err != nil {
			panic(fmt.Sprintf("Failed to download (%s) image! %s", talosImageUrl, err))
		}

		m.Logger.Successf("Download saved to: %s", resp.Filename)

		s.IsDone = true
	}()
	return nil
}

func RunClusterBootstrapStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	go func() {
		m.Logger.Debug("RunClusterBootstrapStepInBackground")
		endpoint := state.ConfigBundle.ControlPlaneCfg.Cluster().Endpoint() //get machineconfig cluster endpoint
		talosApiClient := talos.CreateMachineryClientFromTalosconfig(state.ConfigBundle.TalosConfig())
		m.Logger.Infof("Executing bootstrap with clustername %s and endpoint %s....", bootstrapInfos.TalosInfo.ClusterName, endpoint)
		err := talos.ExecuteBootstrap(talosApiClient)
		if err != nil {
			m.Logger.Errorf("Failed to execute bootstrap: %s", err)
		}
		m.Logger.Success("Bootstrap request Succeeded!")

		m.Logger.Info("Waiting for Kubernetes installation to finish and API to be available...")

		//RunDetailedClusterHealthCheck(talosApiClient, m.Logger)
		talos.RunBasicClusterHealthCheck(err, talosApiClient, m.Logger)
		m.Logger.Success("Cluster health check succeeded!")

		kubeconfig, err = talosApiClient.Kubeconfig(context.Background())
		if err != nil {
			m.Logger.Errorf("Failed to get kubeconfig: %v", err)
			panic(err)
		}

		err = os.WriteFile("kubeconfig", kubeconfig, 0600)

		if err != nil {
			m.Logger.Errorf("Failed to write kubeconfig: %v", err)
			panic(err)
		}

		m.Logger.Successf("Wrote kubeconfig to ./kubeconfig")
		m.Logger.Success("Your cluster is ready! You may now use kubectl to interact with the cluster")

		s.IsDone = true
	}()
	return nil
}

func RunArgoStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	go func() {
		m.Logger.Debug("RunArgoStepInBackground")
		DeployArgoCD(m.Logger)
		s.IsDone = true
	}()
	return nil
}

func RunPortalStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	go func() {
		m.Logger.Debug("RunPortalStepInBackground")
		CreateProviderSecrets(m.Logger)
		s.IsDone = true
	}()
	return nil
}

func DeployArgoCD(loggerRef *tui.UILogger) {
	loggerRef.Info("Setting up helm...")
	helmClient, err := helm.SetupHelmClient(loggerRef, kubeconfig)
	if err != nil {
		loggerRef.Errorf("Failed to setup helm client: %s", err)
		return
	}

	loggerRef.Infof("Deploying ArgoCD...")
	release, err := helm.HelmInstallArgo(helmClient)
	if err != nil {
		loggerRef.Errorf("Failed to deploy ArgoCD: %s", err)
		return
	}
	loggerRef.Successf("Successfully Installed release %s in namespace %s ; Notes:%s\n", release.Name, release.Namespace, release.Info.Notes)
}

func CreateProviderSecrets(loggerRef *tui.UILogger) {
	// Apply provider secrets
	k8sClient, err := k8s.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		loggerRef.Errorf("Failed to create Kubernetes client: %s", err)
	} else {
		ctx := context.Background()

		// Create GCP service account secret
		if gcpConfig != nil {
			loggerRef.Info("Creating GCP service account secret...")
			err = gcpConfig.CreateOrUpdateSecret(ctx, k8sClient, "stolos-system", "gcp-service-account")
			if err != nil {
				loggerRef.Errorf("Failed to create GCP secret: %s", err)
			} else {
				loggerRef.Success("GCP service account secret created successfully")
			}
		}

		// Create GitHub credentials secret
		if githubConfig != nil {
			loggerRef.Info("Creating GitHub credentials secret...")
			err = githubConfig.CreateOrUpdateSecret(ctx, k8sClient, "stolos-system", "github-credentials")
			if err != nil {
				loggerRef.Errorf("Failed to create GitHub secret: %s", err)
			} else {
				loggerRef.Success("GitHub credentials secret created successfully")
			}
		}
	}
}

func StartMachineconfigServerInBackground(model *tui.Model, addr string) {
	model.Logger.Infof("Starting HTTP Machineconfig Server on %s …", addr)
	go func() {
		if err := configserver.StartConfigServer(model, addr, doRestoreProgress, &saveState, bootstrapInfos); err != nil {
			model.Logger.Errorf("Config server stopped: %v", err)
		}
	}()
}

// Utils

// GetOutboundIP Get preferred outbound ip of this machine
// Ref: https://stackoverflow.com/a/37382208
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

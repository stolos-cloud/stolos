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

	"github.com/cavaliergopher/grab/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siderolabs/image-factory/pkg/schematic"
)

var bootstrapInfos = &state.BootstrapInfo{}
var doRestoreProgress = false
var didReadConfig = false

// var tui.Steps []tui.Step
var kubeconfig []byte
var saveState state.SaveState
var gcpConfig *gcp.Config
var githubConfig *github.Config

func main() {

	tui.RegisterDefaultFunc("GetOutboundIP", GetOutboundIP)

	_, err := os.Stat("bootstrap-state.json")

	if !(errors.Is(err, os.ErrNotExist)) {
		saveState = marshal.ReadStateFromJSON()
		bootstrapInfos = &saveState.BootstrapInfo
		didReadConfig = true
		doRestoreProgress = true
	}

	_, err = os.Stat("bootstrap-config.json")
	if !(errors.Is(err, os.ErrNotExist)) {
		marshal.ReadBootstrapInfos("bootstrap-config.json", bootstrapInfos)
		didReadConfig = true
	}

	step1 := tui.Step{
		Name:        "TalosInfo",
		Title:       "1) Basic Information and Image Factory",
		Kind:        tui.StepForm,
		Fields:      tui.CreateFieldsForStruct[state.TalosInfo](),
		IsDone:      true,
		AutoAdvance: false,
	}

	if didReadConfig {
		step1.AutoAdvance = true
	}

	step1_1 := tui.Step{
		Name:        "CreateRepo",
		Title:       "1.1) Creating Repository...",
		Kind:        tui.StepSpinner,
		Body:        "Creating github repository...",
		AutoAdvance: true,
	}

	step1_2 := tui.Step{
		Name:        "GenerateISO",
		Title:       "1.2) Generate Talos Image...",
		Kind:        tui.StepSpinner,
		Body:        "Generating talos image via image factory...",
		AutoAdvance: true,
	}

	step2 := tui.Step{
		Name:        "Boot",
		Title:       "2) Boot",
		Kind:        tui.StepPlain,
		Body:        "Note: The first node that accesses the config server will be configured as a Kubernetes Control Plane",
		AutoAdvance: true,
	}

	step21 := tui.Step{
		Name:        "WaitControlPlane",
		Title:       "2.1) Waiting for First Node (Control Plane)",
		Kind:        tui.StepSpinner,
		Body:        "Waiting for the first node to request machineconfig...",
		AutoAdvance: true,
	}

	step22 := tui.Step{
		Name:        "WaitWorkers",
		Title:       "2.2) Waiting for three worker nodes…",
		Kind:        tui.StepSpinner,
		Body:        "Generating worker base machine config and waiting for 3 workers to fetch their configs…",
		AutoAdvance: false,
		IsDone:      true,
	}

	step23 := tui.Step{
		Name:        "Bootstrap",
		Title:       "2.3) Executing bootstrap…",
		Kind:        tui.StepSpinner,
		Body:        "Bootstrapping the cluster…",
		AutoAdvance: true,
	}

	step24 := tui.Step{
		Name:        "Kubernetes",
		Title:       "2.4) Deploying ArgoCD and the WebUI",
		Kind:        tui.StepSpinner,
		Body:        "Deploying ArgoCD via Helm. ArgoCD will deploy the WebUI.",
		AutoAdvance: false,
	}

	tui.Steps = []tui.Step{step1, step1_1, step1_2, step2, step21, step22, step23, step24}
	p, logger := tui.NewWizard(tui.Steps)
	logger.Infof("Authenticating github client id %s", github.GithubClientId)

	tui.Steps[1].OnEnter = RunRepoGCPStepInBackground

	tui.Steps[2].OnEnter = RunClusterBootupStepInBackground

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	tui.Steps[3].OnEnter = RunStartMachineconfigServerStep

	// Step 2.1 - Waiting for first node
	tui.Steps[4].OnEnter = nil

	// Step 2.2: Waiting for worker nodes
	tui.Steps[5].OnEnter = nil

	// Step 2.3: Bootstrap
	tui.Steps[6].OnEnter = RunClusterBootstrapStepInBackground

	tui.Steps[7].OnEnter = RunKubernetesStepInBackground

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

func RunRepoGCPStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	if !didReadConfig {
		_, talosInfoStep := tui.FindStepByName(m, "TalosInfo")
		talosInfo, err := tui.RetrieveStructFromFields[state.TalosInfo](talosInfoStep.Fields)
		if err != nil {
			m.Logger.Errorf("Error getting TalosInfo from form fields: %v", err)
			panic(err)
		}
		bootstrapInfos.TalosInfo = *talosInfo
	}

	m.Logger.Infof("Read bootstrap infos from file, clusterName: %s", bootstrapInfos.TalosInfo.ClusterName)

	go func() {
		// Setup OAuth server
		oauthServer := oauth.NewServer("9999", m.Logger)

		// Register providers
		if gcp.GCPClientId != "" && gcp.GCPClientSecret != "" {
			gcpProvider := oauth.NewGCPProvider(gcp.GCPClientId, gcp.GCPClientSecret)
			oauthServer.RegisterProvider(gcpProvider)
		}

		githubProvider := oauth.NewGitHubProvider(github.GithubClientId, github.GithubClientSecret)
		oauthServer.RegisterProvider(githubProvider)

		ctx := context.Background()
		if err := oauthServer.Start(ctx); err != nil {
			m.Logger.Errorf("skipping, oauth server start failed: %v", err)
			tui.Steps[1].IsDone = true
			return
		}
		defer oauthServer.Stop(ctx)

		githubToken, err := oauthServer.Authenticate(ctx, "GitHub")
		if err != nil {
			m.Logger.Errorf("skipping, oauth server authenticate failed: %v", err)
			tui.Steps[1].IsDone = true
			return
		}

		// Create GitHub client and initialize repository
		githubClient := github.NewClient(githubToken)
		githubBootstrapInfo := &github.GitHubInfo{
			RepoName:       bootstrapInfos.GitHubInfo.RepoName,
			RepoOwner:      bootstrapInfos.GitHubInfo.RepoOwner,
			BaseDomain:     bootstrapInfos.GitHubInfo.BaseDomain,
			LoadBalancerIP: bootstrapInfos.GitHubInfo.LoadBalancerIP,
		}

		_, err = githubClient.InitRepo(githubBootstrapInfo, false)
		if err != nil {
			m.Logger.Errorf("github init repo failed: %v", err)
		}

		// Create GitHub config for backend
		githubConfig = github.NewConfig(githubToken, bootstrapInfos.GitHubInfo.RepoOwner, bootstrapInfos.GitHubInfo.RepoName)

		m.Logger.Infof("Repo initialized: https://github.com/%s/%s.git", bootstrapInfos.GitHubInfo.RepoOwner, bootstrapInfos.GitHubInfo.RepoName)

		if gcp.GCPClientId != "" && gcp.GCPClientSecret != "" {
			gcpToken, err := oauthServer.Authenticate(ctx, "GCP")
			if err != nil {
				m.Logger.Errorf("Failed to authenticate with GCP: %v", err)
			} else {
				// Create GCP service account
				gcpConfig, err = gcp.CreateServiceAccountWithOAuth(
					ctx,
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
			}
		} else {
			m.Logger.Infof("GCP OAuth credentials not provided, skipping GCP service account creation")
		}

		tui.Steps[1].IsDone = true
	}()
	return nil
}

func RunStartMachineconfigServerStep(m *tui.Model, s *tui.Step) tea.Cmd {
	m.Logger.Infof("Cluster: %s", bootstrapInfos.TalosInfo.ClusterName)
	addr := bootstrapInfos.TalosInfo.HTTPHostname + ":" + bootstrapInfos.TalosInfo.HTTPPort
	StartMachineconfigServerInBackground(m.Logger, addr)
	s.IsDone = true
	return nil
}

func RunClusterBootupStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	if doRestoreProgress {
		m.Logger.Info("State file found, skipping to tui.Steps[5]")
		addr := bootstrapInfos.TalosInfo.HTTPHostname + ":" + bootstrapInfos.TalosInfo.HTTPPort
		StartMachineconfigServerInBackground(m.Logger, addr)
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

func RunKubernetesStepInBackground(m *tui.Model, s *tui.Step) tea.Cmd {
	go func() {
		m.Logger.Debug("RunKubernetesStepInBackground")
		CreateProviderSecrets(m.Logger)
		DeployArgoCD(m.Logger)
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

func StartMachineconfigServerInBackground(loggerRef *tui.UILogger, addr string) {
	loggerRef.Infof("Starting HTTP Machineconfig Server on %s …", addr)
	go func() {
		if err := configserver.StartConfigServer(loggerRef, addr, doRestoreProgress, &saveState, bootstrapInfos); err != nil {
			loggerRef.Errorf("Config server stopped: %v", err)
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

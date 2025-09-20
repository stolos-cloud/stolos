// main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/stolos-cloud/stolos-bootstrap/internal/configserver"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/github"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/helm"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/marshal"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"

	"github.com/cavaliergopher/grab/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siderolabs/image-factory/pkg/schematic"
)

var bootstrapInfos *state.BootstrapInfo
var doRestoreProgress = false
var didReadConfig = false

// var tui.Steps []tui.Step
var kubeconfig []byte
var saveState state.SaveState

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
		Title:       "1) Basic Information and Image Factory",
		Kind:        tui.StepForm,
		Fields:      tui.CreateFieldsForStruct[state.BootstrapInfo](),
		IsDone:      true,
		AutoAdvance: false,
	}

	if didReadConfig {
		step1.AutoAdvance = true
	}

	step1_1 := tui.Step{
		Title:       "1.1) Creating Repository...",
		Kind:        tui.StepSpinner,
		Body:        "Creating github repository...",
		AutoAdvance: true,
	}

	step1_2 := tui.Step{
		Title:       "1.2) Generate Talos Image...",
		Kind:        tui.StepSpinner,
		Body:        "Generating talos image via image factory...",
		AutoAdvance: true,
	}

	step2 := tui.Step{
		Title:       "2) Boot",
		Kind:        tui.StepPlain,
		Body:        "Note: The first node that accesses the config server will be configured as a Kubernetes Control Plane",
		AutoAdvance: true,
	}

	step21 := tui.Step{
		Title:       "2.1) Waiting for First Node (Control Plane)",
		Kind:        tui.StepSpinner,
		Body:        "Waiting for the first node to request machineconfig...",
		AutoAdvance: true,
	}

	step22 := tui.Step{
		Title:       "2.2) Waiting for three worker nodes…",
		Kind:        tui.StepSpinner,
		Body:        "Generating worker base machine config and waiting for 3 workers to fetch their configs…",
		AutoAdvance: false,
		IsDone:      true,
	}

	step23 := tui.Step{
		Title:       "2.3) Executing bootstrap…",
		Kind:        tui.StepSpinner,
		Body:        "Bootstrapping the cluster…",
		AutoAdvance: true,
	}

	step24 := tui.Step{
		Title:       "2.4) Deploying ArgoCD and the WebUI",
		Kind:        tui.StepSpinner,
		Body:        "Deploying ArgoCD via Helm. ArgoCD will deploy the WebUI.",
		AutoAdvance: false,
	}

	tui.Steps = []tui.Step{step1, step1_1, step1_2, step2, step21, step22, step23, step24}
	p, logger := tui.NewWizard(tui.Steps)
	loggerRef := logger
	logger.Infof("Authenticating github client id %s", github.GithubClientId)

	tui.Steps[1].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {

			var err error
			if !didReadConfig {
				bootstrapInfos, err = tui.RetrieveStructFromFields[state.BootstrapInfo](step1.Fields)
			} else {
				loggerRef.Infof("Read bootstrap infos from file, clusterName: %s", bootstrapInfos.ClusterName)
			}

			if err != nil {
				panic(err)
			}

			githubClient, err := github.AuthenticateGithubClient(loggerRef)
			if err != nil {
				panic(err)
			}
			_, err = github.InitRepo(githubClient, bootstrapInfos, false)

			loggerRef.Successf("Repo initialized: https://github.com/%s/%s.git", bootstrapInfos.RepoOwner, bootstrapInfos.RepoName)

			tui.Steps[1].IsDone = true
			return nil
		}
	}
	tui.Steps[2].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {

			if doRestoreProgress {
				loggerRef.Info("State file found, skipping to tui.Steps[5]")
				addr := bootstrapInfos.HTTPHostname + ":" + bootstrapInfos.HTTPPort
				StartMachineconfigServerInBackground(loggerRef, addr)
				m.CurrentStepIndex = 5
				return nil
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				talosConfigArg := fmt.Sprintf("talos.config=http://%s:%s/machineconfig?m=${mac}&u=${uuid}", bootstrapInfos.HTTPHostname, bootstrapInfos.HTTPPort)
				kernelArgs := []string{talosConfigArg, bootstrapInfos.TalosExtraArgs}

				loggerRef.Infof("Generating image with kernelParam: %s", talosConfigArg)

				factory := talos.CreateFactoryClient()
				sch := schematic.Schematic{
					Overlay: schematic.Overlay{
						Image: bootstrapInfos.TalosOverlayImage,
						Name:  bootstrapInfos.TalosOverlayName,
						// Options: nil, // ==> Extra YAML settings passed to overlay image.
					},
					Customization: schematic.Customization{
						ExtraKernelArgs: kernelArgs,
					},
				}

				schematicId, _ := factory.SchematicCreate(ctx, sch)
				loggerRef.Infof("Generated schematicId: %s", schematicId)

				talosImageFormat := "iso"
				if bootstrapInfos.TalosArchitecture == "arm64" {
					talosImageFormat = "raw.xz"
				}

				talosImagePath := fmt.Sprintf("metal-%s.%s", bootstrapInfos.TalosArchitecture, talosImageFormat)
				talosImageUrl := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/%s", schematicId, bootstrapInfos.TalosVersion, talosImagePath)
				//loggerRef.Infof("%s", talosImageUrl)

				loggerRef.Infof("Downloading image from %s...", talosImageUrl)
				resp, err := grab.Get(".", talosImageUrl)
				if err != nil {
					panic(fmt.Sprintf("Failed to download (%s) image! %s", talosImageUrl, err))
				}

				loggerRef.Successf("Download saved to: %s", resp.Filename)

				tui.Steps[2].IsDone = true
			}()

			return nil
		}
	}

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	tui.Steps[3].OnEnter = func(m *tui.Model) tea.Cmd {
		loggerRef.Infof("tui.Steps[2]")

		// Read Step 1 values from the tui.Model
		cluster := strings.TrimSpace(bootstrapInfos.ClusterName)
		if cluster == "" {
			cluster = "mycluster"
		}

		httpPort := strings.TrimSpace(bootstrapInfos.HTTPPort)
		if httpPort == "" {
			httpPort = "8080"
		}

		addr := bootstrapInfos.HTTPHostname + ":" + httpPort

		return tea.Batch(
			func() tea.Msg {
				loggerRef.Infof("Cluster: %s", cluster)
				return nil
			},
			func() tea.Msg {
				// Start the server in a goroutine; never block the TUI.
				StartMachineconfigServerInBackground(loggerRef, addr)
				tui.Steps[2].IsDone = true
				return nil
			},
		)
	}

	// Step 2.1 - Waiting for first node
	tui.Steps[4].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("tui.Steps[3]") // Control Plane Step

			// NOTE : IsDone is set in handleControlPlane

			return nil
		}
	}

	// Step 2.2: Waiting for worker nodes
	tui.Steps[5].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("tui.Steps[4]")

			return nil
		}
	}

	// Step 2.3: Bootstrap
	tui.Steps[6].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("tui.Steps[5]")
			endpoint := state.ConfigBundle.ControlPlaneCfg.Cluster().Endpoint() //get machineconfig cluster endpoint

			talosApiClient := talos.CreateMachineryClientFromTalosconfig(state.ConfigBundle.TalosConfig())
			loggerRef.Infof("Executing bootstrap with clustername %s and endpoint %s....", bootstrapInfos.ClusterName, endpoint)
			err = talos.ExecuteBootstrap(talosApiClient)
			if err != nil {
				loggerRef.Errorf("Failed to execute bootstrap: %s", err)
			}
			loggerRef.Success("Bootstrap request Succeeded!")

			loggerRef.Info("Waiting for Kubernetes installation to finish and API to be available...")

			go func() {

				//RunDetailedClusterHealthCheck(talosApiClient, loggerRef)
				talos.RunBasicClusterHealthCheck(err, talosApiClient, loggerRef)
				loggerRef.Success("Cluster health check succeeded!")

				kubeconfig, err = talosApiClient.Kubeconfig(context.Background())
				if err != nil {
					loggerRef.Errorf("Failed to get kubeconfig: %v", err)
					panic(err)
				}

				err = os.WriteFile("kubeconfig", kubeconfig, 0600)

				if err != nil {
					loggerRef.Errorf("Failed to write kubeconfig: %v", err)
					panic(err)
				}

				loggerRef.Successf("Wrote kubeconfig to ./kubeconfig")
				loggerRef.Success("Your cluster is ready! You may now use kubectl to interact with the cluster")

				tui.Steps[5].IsDone = true
			}()

			return nil
		}
	}

	tui.Steps[7].OnEnter = func(m *tui.Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("tui.Steps[7]")

			go func() {
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
			}()

			return nil
		}
	}

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
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

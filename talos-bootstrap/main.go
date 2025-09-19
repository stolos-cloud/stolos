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

	"github.com/cavaliergopher/grab/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/goccy/go-json"
	"github.com/siderolabs/image-factory/pkg/schematic"
)

type BootstrapInfo struct {
	ClusterName       string `json:"ClusterName" field_label:"Cluster Name" field_required:"true" field_default:"mycluster"`
	KubernetesVersion string `json:"KubernetesVersion" field_label:"Kubernetes versions" field_default:"1.34.1"`
	TalosVersion      string `json:"TalosVersion" field_label:"Talos Version (Optional)" field_default:"v1.11.1"`
	TalosArchitecture string `json:"TalosArchitecture" field_label:"Talos architecture" field_default:"amd64" field_required:"true"`
	TalosExtraArgs    string `json:"TalosExtraArgs" field_label:"Extra Linux cmdline args"`
	TalosInstallDisk  string `json:"TalosInstallDisk" field_label:"Talos install disk" field_default:"/dev/sda" field_required:"true"`
	TalosOverlayImage string `json:"TalosOverlayImage" field_label:"Talos Overlay Image (For SBC, ex: siderolabs/sbc-rockchip)"`
	TalosOverlayName  string `json:"TalosOverlayName" field_label:"Talos Overlay Name (For SBC, ex: turingrk1)"`
	HTTPHostname      string `json:"HTTPHostname" field_label:"HTTP Machineconfig Server External Hostname" field_required:"true" field_default_func:"GetOutboundIP"`
	HTTPPort          string `json:"HTTPPort" field_label:"HTTP Machineconfig Server Port" field_required:"true" field_default:"8082"`
	PXEEnabled        string `json:"PXEEnabled" field_label:"PXE Server Enabled (true/false)" field_default:"false"`
	PXEPort           string `json:"PXEPort" field_label:"PXE Server Port (Optional)"`
}

var bootstrapInfos *BootstrapInfo
var doRestoreProgress = false
var didReadConfig = false
var steps []Step

func readBootstrapInfos(filename string) {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configFile, &bootstrapInfos)
	if err != nil {
		panic(err)
	}
}

func saveStateToJSON(logger *UILogger) {
	jsonData, err := json.Marshal(saveState)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
	err = os.WriteFile("talos-bootstrap-state.json", jsonData, 0644)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
	err = SaveSplitConfigBundleFiles(logger, *configBundle)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
}

func readStateFromJSON() {
	stateFile, err := os.ReadFile("talos-bootstrap-state.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(stateFile, &saveState)
	if err != nil {
		panic(err)
	}
	bootstrapInfos = &saveState.BootstrapInfo
	configBundle, err = ReadSplitConfigBundleFiles()
	if err != nil {
		panic(err)
	}
}

func main() {

	RegisterDefaultFunc("GetOutboundIP", GetOutboundIP)

	_, err := os.Stat("talos-bootstrap-state.json")

	if !(errors.Is(err, os.ErrNotExist)) {
		readStateFromJSON()
		didReadConfig = true
		doRestoreProgress = true
	}

	_, err = os.Stat("talos-bootstrap-config.json")
	if !(errors.Is(err, os.ErrNotExist)) {
		readBootstrapInfos("talos-bootstrap-config.json")
		didReadConfig = true
	}

	step1 := Step{
		Title:       "1) Basic Information and Image Factory",
		Kind:        StepForm,
		Fields:      createFieldsForStruct[BootstrapInfo](),
		IsDone:      true,
		AutoAdvance: false,
	}

	if didReadConfig {
		step1.AutoAdvance = true
	}

	step1_1 := Step{
		Title:       "1.1) Generate Talos Image...",
		Kind:        StepSpinner,
		Body:        "Generating talos image via image factory...",
		AutoAdvance: true,
	}

	step2 := Step{
		Title:       "2) Boot",
		Kind:        StepPlain,
		Body:        "Note: The first node that accesses the config server will be configured as a Kubernetes Control Plane",
		AutoAdvance: true,
	}

	step21 := Step{
		Title:       "2.1) Waiting for First Node (Control Plane)",
		Kind:        StepSpinner,
		Body:        "Waiting for the first node to request machineconfig...",
		AutoAdvance: true,
	}

	step22 := Step{
		Title:       "2.2) Waiting for three worker nodes…",
		Kind:        StepSpinner,
		Body:        "Generating worker base machine config and waiting for 3 workers to fetch their configs…",
		AutoAdvance: false,
		IsDone:      true,
	}

	step23 := Step{
		Title:       "2.3) Executing bootstrap…",
		Kind:        StepSpinner,
		Body:        "Bootstrapping the cluster…",
		AutoAdvance: false,
	}

	steps = []Step{step1, step1_1, step2, step21, step22, step23}
	p, logger := NewWizard(steps)
	loggerRef := logger

	steps[1].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {

			var err error
			if !didReadConfig {
				bootstrapInfos, err = retrieveStructFromFields[BootstrapInfo](step1.Fields)
			} else {
				loggerRef.Infof("Read bootstrap infos from file, clusterName: %s", bootstrapInfos.ClusterName)
			}

			if err != nil {
				panic(err)
			}

			if doRestoreProgress {
				loggerRef.Info("State file found, skipping to steps[5]")
				addr := bootstrapInfos.HTTPHostname + ":" + bootstrapInfos.HTTPPort
				StartMachineconfigServerInBackground(loggerRef, addr)
				m.currentStepIndex = 4
				return nil
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				talosConfigArg := fmt.Sprintf("talos.config=http://%s:%s/machineconfig?m=${mac}&u=${uuid}", bootstrapInfos.HTTPHostname, bootstrapInfos.HTTPPort)
				kernelArgs := []string{talosConfigArg, bootstrapInfos.TalosExtraArgs}

				loggerRef.Infof("Generating image with kernelParam: %s", talosConfigArg)

				factory := CreateFactoryClient()
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

				steps[1].IsDone = true
			}()

			return nil
		}
	}

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	steps[2].OnEnter = func(m *Model) tea.Cmd {
		loggerRef.Infof("steps[2]")

		// Read Step 1 values from the model
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
				steps[2].IsDone = true
				return nil
			},
		)
	}

	// Step 2.1 - Waiting for first node
	steps[3].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[3]") // Control Plane Step

			// NOTE : IsDone is set in handleControlPlane

			return nil
		}
	}

	// Step 2.2: Waiting for worker nodes
	steps[4].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[4]")

			return nil
		}
	}

	// Step 2.3: Bootstrap
	steps[5].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[5]")
			endpoint := configBundle.ControlPlaneCfg.Cluster().Endpoint() //get machineconfig cluster endpoint

			talosApiClient := CreateMachineryClientFromTalosconfig(configBundle.TalosConfig())
			loggerRef.Infof("Executing bootstrap with clustername %s and endpoint %s....", bootstrapInfos.ClusterName, endpoint)
			err = ExecuteBootstrap(talosApiClient)
			if err != nil {
				loggerRef.Errorf("Failed to execute bootstrap: %s", err)
			}
			loggerRef.Success("Bootstrap request Succeeded!")

			loggerRef.Info("Waiting for Kubernetes installation to finish and API to be available...")

			go func() {

				RunDetailedClusterHealthCheck(talosApiClient, loggerRef)
				//RunBasicClusterHealthCheck(err, talosApiClient, loggerRef)
				loggerRef.Success("Cluster health check succeeded!")

				kubeconfig, err := talosApiClient.Kubeconfig(context.Background())
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
			}()

			return nil
		}
	}

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

func StartMachineconfigServerInBackground(loggerRef *UILogger, addr string) {
	loggerRef.Infof("Starting HTTP Machineconfig Server on %s …", addr)
	go func() {
		if err := StartConfigServer(loggerRef, addr); err != nil {
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

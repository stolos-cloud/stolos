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

	tea "github.com/charmbracelet/bubbletea"
	schematic "github.com/siderolabs/image-factory/pkg/schematic"
)

// Indices for Step 1 fields
const (
	idxClusterName = iota
	idxTalosVersion
	idxImageOverlay
	idxMCOverlay
	idxHTTPHostname
	idxHTTPPort
	idxPXEEnabled
	idxPXEPort
)

type BootstrapInfo struct {
	ClusterName              string
	TalosVersion             string
	TalosArchitecture        string
	ImageOverlayPath         string
	MachineconfigOverlayPath string
	HTTPHostname             string
	HTTPPort                 string
	PXEEnabled               string
	PXEPort                  string
	KubernetesVersion        string
}

var bootstrapInfos = BootstrapInfo{}

var doRestoreProgress = false

func main() {

	_, err := os.Stat("talos-bootstrap-state.json")
	doRestoreProgress = errors.Is(err, os.ErrNotExist)
	//TODO when true, skip to step 4

	step1 := Step{
		Title: "1) Basic Information and Image Factory",
		Kind:  StepForm,
		// TODO: Bug in Bubbletea causes placeholder not to work - check after package is updated
		Fields: []Field{
			NewTextField("Cluster Name", "my-cluster-01", false),
			NewTextField("Talos Version (Optional)", "", true),
			NewTextField("Custom Image Factory YAML Overlay (Optional)", "", true),
			NewTextField("Custom Machineconfig YAML Overlay (Optional)", "", true),
			NewTextField("HTTP Machineconfig Server External Hostname", "", false),
			NewTextField("HTTP Machineconfig Server Port", "", false),
			NewTextField("PXE Server Enabled (true/false)", "", true),
			NewTextField("PXE Server Port (Optional)", "", true),
		},
	}

	// Default form values:
	step1.Fields[idxTalosVersion].Input.SetValue("v1.11.1")
	step1.Fields[idxClusterName].Input.SetValue("mycluster")
	step1.Fields[idxHTTPHostname].Input.SetValue(GetOutboundIP())
	step1.Fields[idxHTTPPort].Input.SetValue("8082")
	step1.Fields[idxPXEEnabled].Input.SetValue("false")
	step1.Fields[idxPXEPort].Input.SetValue("69")

	step1_1 := Step{
		Title:       "1.1) Generate Talos Image...",
		Kind:        StepSpinner,
		Body:        "Generating talos image via image factory...",
		AutoAdvance: false,
	}

	step2 := Step{
		Title:       "2) Boot",
		Kind:        StepPlain,
		Body:        "Note: The first node that accesses the config server will be configured as a Kubernetes Control Plane",
		AutoAdvance: false,
	}

	step21 := Step{
		Title:       "2.1) Waiting for First Node (Control Plane)",
		Kind:        StepSpinner,
		Body:        "Waiting for the first node to request machineconfig...",
		AutoAdvance: false,
	}

	step22 := Step{
		Title:       "2.2) Waiting for three worker nodes…",
		Kind:        StepSpinner,
		Body:        "Generating worker base machine config and waiting for 3 workers to fetch their configs…",
		AutoAdvance: false,
	}

	step23 := Step{
		Title:       "2.3) Executing bootstrap…",
		Kind:        StepSpinner,
		Body:        "Bootstrapping the cluster…",
		AutoAdvance: false,
	}

	steps := []Step{step1, step1_1, step2, step21, step22, step23}

	p, logger := NewWizard(steps)

	loggerRef := logger

	steps[1].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {

			if doRestoreProgress {
				m.currentStepIndex = 4
				return nil //  TODO : test test skip logic
			}

			step1_MapFormValuesToBootstrapInfos(step1)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			talosConfigArg := fmt.Sprintf("talos.config=http://%s:%s/machineconfig?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}", step1.Fields[idxHTTPHostname].Input.Value(), step1.Fields[idxHTTPPort].Input.Value())
			kernelArgs := append(make([]string, 1), talosConfigArg)

			loggerRef.Infof("Generating image with kernelParam: %s", talosConfigArg)

			// TuringPI : ?arch=arm64&board=turingrk1&extensions=-&platform=metal&target=sbc&version=1.11.1
			// overlay:
			//    image: siderolabs/sbc-rockchip
			//    name: turingrk1
			// customization: {}

			factory := CreateFactoryClient()
			sch := schematic.Schematic{
				Overlay: schematic.Overlay{
					// TODO : Add form options for SBC or just Handle via custom YAML file overlay
					Image: "siderolabs/sbc-rockchip",
					Name:  "turingrk1",
					// Options: nil, // ==> Extra YAML settings passed to overlay image.
				},
				Customization: schematic.Customization{
					ExtraKernelArgs: kernelArgs,
				},
			}

			schematicId, _ := factory.SchematicCreate(ctx, sch)
			loggerRef.Infof("Generated schematicId: %s", schematicId)

			talosImageFormat := "raw.xz"
			talosImagePath := fmt.Sprintf("metal-%s.%s", bootstrapInfos.TalosArchitecture, talosImageFormat)
			talosImageUrl := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/%s", schematicId, bootstrapInfos.TalosVersion, talosImagePath)
			loggerRef.Infof("%s", talosImageUrl)
			// TuringPI RK2 : https://factory.talos.dev/image/df156b82096feda49406ac03aa44e0ace524b7efe4e1f0e144a1e1ae3930f1c0/v1.11.1/metal-arm64.raw.xz

			/*resp, err := grab.Get(".", talosImageUrl)
			if err != nil {
				panic(fmt.Sprintf("Failed to download (%s) image! %s", talosImageUrl, err))
			}

			loggerRef.Infof("Download saved to: %s", resp.Filename)*/

			return nil
		}
	}

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	steps[2].OnEnter = func(m *Model) tea.Cmd {
		loggerRef.Infof("steps[2]")

		// Read Step 1 values from the model
		cluster := strings.TrimSpace(m.steps[0].Fields[idxClusterName].Input.Value())
		if cluster == "" {
			cluster = "mycluster"
		}
		httpEnabled := true
		httpPort := strings.TrimSpace(m.steps[0].Fields[idxHTTPPort].Input.Value())
		if httpPort == "" {
			httpPort = "8080"
		}
		addr := "0.0.0.0:" + httpPort

		return tea.Batch(
			func() tea.Msg {
				loggerRef.Infof("Cluster: %s", cluster)
				return nil
			},
			func() tea.Msg {
				if !httpEnabled {
					loggerRef.Warn("HTTP Machineconfig Server is disabled (enable it in Step 1 to serve /machineconfig)")
					return nil
				}
				//loggerRef.Infof("Starting HTTP Machineconfig Server on %s …", addr)
				// Start the server in a goroutine; never block the TUI.
				go func() {
					if err := StartConfigServer(loggerRef, addr); err != nil {
						loggerRef.Errorf("Config server stopped: %v", err)
					}
				}()
				return nil
			},
		)
	}

	// Step 2.1: show some example log messages upon entering the waiting screen.
	steps[3].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[3]")
			loggerRef.Info("Waiting for first node to hit /machineconfig …")
			loggerRef.Info("Tip: The first requester becomes the Kubernetes Control Plane.")
			return nil
		}
	}

	// Step 2.2: example worker logs
	steps[4].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[4]")
			loggerRef.Info("Generating worker base machine config…")
			loggerRef.Success("Found Worker 1 10.0.0.21 , Responded with worker.machineconfig.yaml")
			loggerRef.Success("Found Worker 2 10.0.0.22 , Responded with worker.machineconfig.yaml")
			loggerRef.Success("Found Worker 3 10.0.0.23 , Responded with worker.machineconfig.yaml")
			loggerRef.Success("3x Workers found ! Execute bootstrap ?")
			return nil
		}
	}

	// Step 2.3: example bootstrap logs (use inputs for $NAME)
	steps[5].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("steps[5]")
			cluster := strings.TrimSpace(m.steps[0].Fields[idxClusterName].Input.Value())
			if cluster == "" {
				cluster = "mycluster"
			}

			talosApiClient := CreateMachineryClientFromTalosconfig(configBundle.TalosConfig())
			ExecuteBootstrap(talosApiClient)

			endpoint := "https://$IP:6443" // placeholder; real value would come from first node IP
			loggerRef.Infof("Executing bootstrap with clustername %s and endpoint %s....", cluster, endpoint)
			loggerRef.Success("Bootstrap Succeeded !")
			loggerRef.Successf("Writing Kubeconfig to %s", "./kubeconfig")
			return nil
		}
	}

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

func step1_MapFormValuesToBootstrapInfos(step1 Step) {
	bootstrapInfos.ClusterName = step1.Fields[idxClusterName].Input.Value()
	bootstrapInfos.TalosVersion = step1.Fields[idxTalosVersion].Input.Value()
	bootstrapInfos.HTTPHostname = step1.Fields[idxHTTPHostname].Input.Value()
	bootstrapInfos.HTTPPort = step1.Fields[idxHTTPPort].Input.Value()
	bootstrapInfos.MachineconfigOverlayPath = step1.Fields[idxMCOverlay].Input.Value()
	bootstrapInfos.ImageOverlayPath = step1.Fields[idxImageOverlay].Input.Value()
	bootstrapInfos.PXEEnabled = step1.Fields[idxImageOverlay].Input.Value()
	bootstrapInfos.PXEPort = step1.Fields[idxPXEPort].Input.Value()
	bootstrapInfos.TalosArchitecture = "arm64"
	bootstrapInfos.KubernetesVersion = "1.34.1"
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

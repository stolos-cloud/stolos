// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/cavaliergopher/grab/v3"
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
	ClusterName              string `field_label:"Cluster Name" field_required:"true" field_default:"mycluster"`
	TalosVersion             string `field_label:"Talos Version (Optional)" field_default:"v1.8.0"`
	ImageOverlayPath         string `field_label:"Custom Image Factory YAML Overlay (Optional)"`
	MachineconfigOverlayPath string `field_label:"Custom Machineconfig YAML Overlay (Optional)"`
	HTTPHostname             string `field_label:"HTTP Machineconfig Server External Hostname" field_required:"true"`
	HTTPPort                 string `field_label:"HTTP Machineconfig Server Port" field_required:"true" `
	PXEEnabled               string `field_label:"PXE Server Enabled (true/false)" field_default:"false"`
	PXEPort                  string `field_label:"PXE Server Port (Optional)"`
}

var bootstrapInfos = &BootstrapInfo{}

func main() {

	// Step 1
	step1 := Step{
		Title: "1) Basic Information and Image Factory",
		Kind:  StepForm,
		// TODO: Bug in Bubbletea causes placeholder not to work - check after package is updated
		Fields: createFieldsForStruct[BootstrapInfo](),
	}

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
		Title: "2.1) Waiting for First Node (Control Plane)",
		Kind:  StepSpinner,
		Body:  "Waiting for the first node to request machineconfig...",
	}

	step22 := Step{
		Title: "2.2) Waiting for three worker nodes…",
		Kind:  StepSpinner,
		Body:  "Generating worker base machine config and waiting for 3 workers to fetch their configs…",
	}

	step23 := Step{
		Title: "2.3) Executing bootstrap…",
		Kind:  StepSpinner,
		Body:  "Bootstrapping the cluster…",
	}

	steps := []Step{step1, step1_1, step2, step21, step22, step23}

	p, logger := NewWizard(steps)

	loggerRef := logger

	steps[1].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {

			var err error
			bootstrapInfos, err = retrieveStructFromFields[BootstrapInfo](step1.Fields)
			if err != nil {
				panic(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			talosConfigArg := fmt.Sprintf("talos.config=http://%s:%s/talosconfig?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}", step1.Fields[idxHTTPHostname].Input.Value(), step1.Fields[idxHTTPPort].Input.Value())
			kernelArgs := append(make([]string, 1), talosConfigArg)

			loggerRef.Infof("Generating image with kernelParam: %s", talosConfigArg)

			factory := CreateFactoryClient()
			sch := schematic.Schematic{

				Customization: schematic.Customization{
					ExtraKernelArgs:  kernelArgs,
					Meta:             nil,
					SystemExtensions: schematic.SystemExtensions{},
					SecureBoot:       schematic.SecureBootCustomization{},
				},
			}

			schematicId, _ := factory.SchematicCreate(ctx, sch)

			resp, err := grab.Get(".", fmt.Sprintf("https://factory.talos.dev/image/%s/%s/%s", schematicId))
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Download saved to", resp.Filename)

			loggerRef.Info(schematicId)
			return m.advanceCmd()
		}
	}

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	steps[2].OnEnter = func(m *Model) tea.Cmd {
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
			loggerRef.Info("Spinner active. Waiting for first node to hit /machineconfig …")
			loggerRef.Info("Tip: The first requester becomes the Kubernetes Control Plane.")
			return nil
		}
	}

	// Step 2.2: example worker logs
	steps[4].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
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
			cluster := strings.TrimSpace(m.steps[0].Fields[idxClusterName].Input.Value())
			if cluster == "" {
				cluster = "mycluster"
			}
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

// Utils

// Get preferred outbound ip of this machine
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

// main.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Indices for Step 1 fields
const (
	idxClusterName = iota
	idxTalosVersion
	idxImageOverlay
	idxMCOverlay
	idxHTTPEnabled
	idxHTTPPort
	idxPXEEnabled
	idxPXEPort
)

func main() {
	step1 := Step{
		Title: "1) Basic Information and Image Factory",
		Kind:  StepForm,
		Fields: []Field{
			NewTextField("Cluster Name", "mycluster", false),
			NewTextField("Talos Version (Optional)", "v1.7.x", true),
			NewTextField("Custom Image Factory YAML Overlay (Optional)", "path/to/image-factory-overlay.yaml", true),
			NewTextField("Custom Machineconfig YAML Overlay (Optional)", "path/to/machineconfig-overlay.yaml", true),
			NewTextField("HTTP Machineconfig Server Enabled (true/false)", "true", false),
			NewTextField("HTTP Machineconfig Server Port", "8080", false),
			NewTextField("PXE Server Enabled (true/false)", "false", true),
			NewTextField("PXE Server Port (Optional)", "69", true),
		},
	}

	// Default form values:
	step1.Fields[idxClusterName].Input.SetValue("mycluster")
	step1.Fields[idxHTTPEnabled].Input.SetValue("true")
	step1.Fields[idxHTTPPort].Input.SetValue("8080")
	step1.Fields[idxPXEEnabled].Input.SetValue("false")
	step1.Fields[idxPXEPort].Input.SetValue("69")

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

	steps := []Step{step1, step2, step21, step22, step23}

	p, logger := NewWizard(steps)

	loggerRef := logger

	// Step 2 (Boot): start HTTP server as soon as we enter the step.
	steps[1].OnEnter = func(m *Model) tea.Cmd {
		// Read Step 1 values from the model
		cluster := strings.TrimSpace(m.steps[0].Fields[idxClusterName].Input.Value())
		if cluster == "" {
			cluster = "mycluster"
		}
		httpEnabled := parseBool(m.steps[0].Fields[idxHTTPEnabled].Input.Value())
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
				loggerRef.Infof("Starting HTTP Machineconfig Server on %s …", addr)
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
	steps[2].OnEnter = func(m *Model) tea.Cmd {
		return func() tea.Msg {
			loggerRef.Info("Spinner active. Waiting for first node to hit /machineconfig …")
			loggerRef.Info("Tip: The first requester becomes the Kubernetes Control Plane.")
			return nil
		}
	}

	// Step 2.2: example worker logs
	steps[3].OnEnter = func(m *Model) tea.Cmd {
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
	steps[4].OnEnter = func(m *Model) tea.Cmd {
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

// parseBool fuzzy
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	case "0", "f", "false", "no", "n", "off":
		return false
	default:
		// Try to parse as int in case someone types a port number accidentally :)
		if i, err := strconv.Atoi(s); err == nil {
			return i != 0
		}
		return false
	}
}

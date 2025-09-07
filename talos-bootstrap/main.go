package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

/* --------------------------------------------------------------------------------
   Steps
-------------------------------------------------------------------------------- */

type stepID int

const (
	step_1_BasicAndImageFactory stepID = iota
	step_2_Boot
	step_2_1_WaitFirstCP
	step_2_2_WaitWorkers
	step_2_3_ExecuteBootstrap
)

type stepMeta struct {
	id       stepID
	title    string
	subtitle string
}

var steps = []stepMeta{
	{step_1_BasicAndImageFactory, "1) Basic Information and Image Factory", "Cluster, Talos version, overlays, HTTP/PXE server setup"},
	{step_2_Boot, "2) Boot", "Note: First node to hit config server becomes a Control Plane"},
	{step_2_1_WaitFirstCP, "2.1) Waiting for First Node (Control Plane)", "Spinner"},
	{step_2_2_WaitWorkers, "2.2) Waiting for three worker nodes…", "Spinner"},
	{step_2_3_ExecuteBootstrap, "2.3) Executing bootstrap…", "Spinner"},
}

/* --------------------------------------------------------------------------------
   Shared State
-------------------------------------------------------------------------------- */

type wizardState struct {
	ClusterName              string
	TalosVersion             string
	ImageFactoryOverlayPath  string
	MachineConfigOverlayPath string
	HTTPEnabled              bool
	HTTPPort                 string
	PXEEnabled               bool
	PXEPort                  string

	FirstControlPlaneIP string
	AnyFilePath         string
	Endpoint            string
}

/* --------------------------------------------------------------------------------
   Root Model (wizard shell)
-------------------------------------------------------------------------------- */

type model struct {
	width, height int
	activeIdx     int
	pages         []page
	completed     map[stepID]bool
	wizState      wizardState
}

func (m *model) vars() map[string]string {
	// Provide variables for interpolation in spinner pages
	name := m.wizState.ClusterName
	if strings.TrimSpace(name) == "" {
		name = "$NAME"
	}
	endpoint := m.wizState.Endpoint
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "$ENDPOINT"
	}
	ip := m.wizState.FirstControlPlaneIP
	if strings.TrimSpace(ip) == "" {
		ip = "$IP"
	}
	file := m.wizState.AnyFilePath
	if strings.TrimSpace(file) == "" {
		file = "$FILE"
	}
	return map[string]string{
		"NAME":     name,
		"ENDPOINT": endpoint,
		"IP":       ip,
		"FILE":     file,
	}
}

func newModel() *model {
	m := &model{
		activeIdx: 0,
		completed: map[stepID]bool{},
		wizState: wizardState{
			// Safe defaults for placeholders
			AnyFilePath: "/tmp/output.yaml",
			// Endpoint intentionally left as "$ENDPOINT" until real logic populates it.
		},
	}

	/* ------------------------- Step 1: Form ------------------------- */

	step1Form := newFormPage(
		"Basic Information and Image Factory",
		"Provide cluster basics, optional overlays, and HTTP/PXE server configuration.",
		styleDim.Render("Note: Advanced network and disk configurations must be specified in the Machineconfig Overlay."),
		[]formField{
			{Key: "cluster_name", Label: "Cluster Name", Required: true, Help: "e.g., protos-talos", Input: newTextInput("cluster name", "", 40)},
			{Key: "talos_version", Label: "Talos Version (optional)", Required: false, Help: "e.g., v1.7.x", Input: newTextInput("v1.7.x", "", 20)},
			{Key: "image_overlay", Label: "Custom Image Factory YAML Overlay (optional)", Required: false, Input: newTextInput("/path/to/imagefactory-overlay.yaml", "", 52)},
			{Key: "mc_overlay", Label: "Custom Machineconfig YAML Overlay (optional)", Required: false, Help: "Put advanced network & disk configs here", Input: newTextInput("/path/to/machineconfig-overlay.yaml", "", 52)},

			// HTTP Machineconfigserver (Mandatory) / PXE (Optional)
			{Key: "http_port", Label: "HTTP Machineconfig Server Port", Required: true, Input: newTextInput("8080", "8080", 8), Validation: validatePort},
			{Key: "pxe_enabled", Label: "Enable PXE Server (true/false)", Required: false, Input: newTextInput("false", "false", 8), Validation: validateBool},
			{Key: "pxe_port", Label: "PXE Server Port (optional)", Required: false, Help: "Ignored if PXE disabled", Input: newTextInput("69", "", 8), Validation: optionalPort},
		},
	)

	/* ------------------------- Step 2: Info ------------------------- */

	step2Info := newInfoPage(
		"Boot",
		"Launch nodes with the generated/served configs",
		"Note: The first node that accesses the config server will be configured as a Kubernetes Control Plane.\n\nPress Enter to continue.",
	)

	/* ------------------------- Step 2.1: Spinner ------------------------- */

	step2_1 := newSpinnerPage(
		"Waiting for First Node (Control Plane)",
		"HTTP Machineconfig server listening…",
		[]string{
			"HTTP Request from $IP!",
			"Generating controlplane machineconfig on the fly…",
			"Saved to $FILE.",
			"Responding to HTTP request with controlplane machineconfig…",
			"Generating talosconfig with endpoint https://$IP:6443 on the fly…",
			"Talosconfig saved to $FILE.",
		},
		m.vars,
	)

	/* ------------------------- Step 2.2: Spinner ------------------------- */

	step2_2 := newSpinnerPage(
		"Waiting for three worker nodes…",
		"Serving worker machineconfigs…",
		[]string{
			"Generating worker base machine config…",
			"Found Worker 1 $IP , Responded with worker.machineconfig.yaml",
			"Found Worker 2 $IP , Responded with worker.machineconfig.yaml",
			"Found Worker 3 $IP , Responded with worker.machineconfig.yaml",
			"3x Workers found ! Execute bootstrap ?",
		},
		m.vars,
	)

	/* ------------------------- Step 2.3: Spinner ------------------------- */

	step2_3 := newSpinnerPage(
		"Executing bootstrap…",
		"Orchestrating Talos bootstrap",
		[]string{
			"Executing bootstrap with clustername $NAME and endpoint $ENDPOINT…",
			"Bootstrap Succeeded!",
			"Writing Kubeconfig to $FILE",
		},
		m.vars,
	)

	m.pages = []page{
		step1Form,
		step2Info,
		step2_1,
		step2_2,
		step2_3,
	}

	return m
}

/* --------------------------------------------------------------------------------
   Validation helpers
-------------------------------------------------------------------------------- */

func validatePort(s string) (bool, string) {
	if strings.TrimSpace(s) == "" {
		return false, "port is required"
	}
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 {
		return false, "invalid port (must be 1-65535)"
	}
	return true, ""
}

func optionalPort(s string) (bool, string) {
	if strings.TrimSpace(s) == "" {
		return true, ""
	}
	return validatePort(s)
}

func validateBool(s string) (bool, string) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "true" || v == "false" {
		return true, ""
	}
	return false, "must be true or false"
}

func requireTrue(s string) (bool, string) {
	ok, msg := validateBool(s)
	if !ok {
		return ok, msg
	}
	if strings.ToLower(strings.TrimSpace(s)) != "true" {
		return false, "HTTP Machineconfig server must be enabled"
	}
	return true, ""
}

/* --------------------------------------------------------------------------------
   main
-------------------------------------------------------------------------------- */

func main() {
	if err := tea.NewProgram(newModel(), tea.WithAltScreen()).Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

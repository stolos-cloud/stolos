package talos

type TalosInfo struct {
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

type Machines struct {
	ControlPlanes map[string][]byte `json:"ControlPlanes"` // map IP : Hostname
	Workers       map[string][]byte `json:"Workers"`       // map IP : Hostname
}

type ServerConfig struct {
	Role        int `json:"Role" field_label:"Kubernetes role"`
	InstallDisk int `json:"InstallDisk" field_required:"true" field_label:"Install disk" `
}

type MachinesDisks map[string]string

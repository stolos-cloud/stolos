package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NodeStatus string

const (
	StatusPending      NodeStatus = "pending"
	StatusProvisioning NodeStatus = "provisioning"
	StatusActive       NodeStatus = "active"
	StatusFailed       NodeStatus = "failed"
)

var ValidNodeStatuses = map[NodeStatus]bool{
	StatusPending:      true,
	StatusProvisioning: true,
	StatusActive:       true,
	StatusFailed:       true,
}

type Node struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
	Name         string         `json:"name" gorm:"not null;uniqueIndex"`
	Status       NodeStatus     `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	Role         string         `json:"role" gorm:"not null;default:'worker'"` // worker, control-plane
	Labels       string         `json:"labels"`                                // JSON string for labels
	Architecture string         `json:"architecture" gorm:"not null"`          // amd64, arm64
	Provider     string         `json:"provider" gorm:"not null"`              // onprem, gcp
	IPAddress    string         `json:"ip_address"`
	MACAddress   string         `json:"mac_address"`
	InstanceID   string         `json:"instance_id,omitempty"` // GCP instance ID
	ClusterID    uuid.UUID      `json:"cluster_id" gorm:"type:uuid;index"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (n *Node) BeforeCreate(tx *gorm.DB) error {
	if n.ID == (uuid.UUID{}) {
		n.ID = uuid.New()
	}
	return nil
}

type Cluster struct {
	ID    uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name  string    `json:"name" gorm:"not null;uniqueIndex"`
	Nodes []Node    `json:"nodes" gorm:"foreignKey:ClusterID"`

	// Talos configuration fields
	TalosVersion string `json:"talos_version,omitempty"`
	KubeVersion  string `json:"kube_version,omitempty"`

	TalosConfig []byte `json:"-" gorm:"type:bytea"`

	// Machine config templates for provisioning new nodes
	ControlPlaneConfig []byte `json:"-" gorm:"type:bytea"` // controlplane.yaml
	WorkerConfig       []byte `json:"-" gorm:"type:bytea"` // worker.yaml

	// Full config bundle backup
	ConfigBundle []byte `json:"-" gorm:"type:bytea"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == (uuid.UUID{}) {
		c.ID = uuid.New()
	}
	return nil
}

type GCPConfig struct {
	ID                    uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
	ProjectID             string         `json:"project_id" gorm:"not null"`
	BucketName            string         `json:"bucket_name" gorm:"not null"`
	ServiceAccountEmail   string         `json:"service_account_email" gorm:"not null"`
	ServiceAccountKeyJSON string         `json:"-" gorm:"type:text"`
	Region                string         `json:"region" gorm:"default:'us-central1'"`
	IsConfigured          bool           `json:"is_configured" gorm:"default:false"`
	InfrastructureStatus  string         `json:"infrastructure_status" gorm:"default:'unconfigured'"` // unconfigured, pending, initializing, ready, failed
	TalosVersion          string         `json:"talos_version" gorm:"default:'v1.11.1'"`
	TalosImageAMD64       string         `json:"talos_image_amd64"`
	TalosImageARM64       string         `json:"talos_image_arm64"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

func (g *GCPConfig) TerraformEnvVars() map[string]string {
	return map[string]string{
		"GOOGLE_CREDENTIALS": g.ServiceAccountKeyJSON,
		"GOOGLE_PROJECT":     g.ProjectID,
	}
}

func (g *GCPConfig) BeforeCreate(tx *gorm.DB) error {
	if g.ID == (uuid.UUID{}) {
		g.ID = uuid.New()
	}
	return nil
}

type GCPResources struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
	LastUpdated time.Time      `json:"last_updated" gorm:"not null"`
	Resources   datatypes.JSON `json:"resources" gorm:"type:jsonb;not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (g *GCPResources) BeforeCreate(tx *gorm.DB) error {
	if g.ID == (uuid.UUID{}) {
		g.ID = uuid.New()
	}
	return nil
}

type GitOpsConfig struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
	RepoOwner    string         `json:"repo_owner" gorm:"not null"`
	RepoName     string         `json:"repo_name" gorm:"not null"`
	Branch       string         `json:"branch" gorm:"not null;default:'main'"`
	WorkingDir   string         `json:"working_dir" gorm:"not null;default:'terraform'"`
	Username     string         `json:"username" gorm:"not null;default:'Stolos Bot'"`
	Email        string         `json:"email" gorm:"not null;default:'bot@stolos.cloud'"`
	IsConfigured bool           `json:"is_configured" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (g *GitOpsConfig) BeforeCreate(tx *gorm.DB) error {
	if g.ID == (uuid.UUID{}) {
		g.ID = uuid.New()
	}
	// Set defaults
	if g.Branch == "" {
		g.Branch = "main"
	}
	if g.WorkingDir == "" {
		g.WorkingDir = "terraform"
	}
	if g.Username == "" {
		g.Username = "Stolos Bot"
	}
	if g.Email == "" {
		g.Email = "bot@stolos.cloud"
	}
	return nil
}

type ISORequest struct {
	Architecture    string   `json:"architecture"`
	TalosVersion    string   `json:"talos_version"`
	ExtraKernelArgs []string `json:"extra_kernel_args,omitempty"`
	OverlayImage    string   `json:"overlay_image,omitempty"`
	OverlayName     string   `json:"overlay_name,omitempty"`
}

type ISOResponse struct {
	DownloadURL  string `json:"download_url"`
	Filename     string `json:"filename"`
	SchematicID  string `json:"schematic_id"`
	TalosVersion string `json:"talos_version"`
	Architecture string `json:"architecture"`
}

type OnPremNodeProvisionRequest struct {
	Nodes []OnPremNodeProvisionConfig `json:"nodes" binding:"required"`
}

type OnPremNodeProvisionConfig struct {
	NodeID      uuid.UUID `json:"node_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role        string    `json:"role" binding:"required" example:"worker"`
	Labels      []string  `json:"labels" example:"zone=us-east,type=compute"`
	InstallDisk string    `json:"install_disk" example:"/dev/sda"`
}

type NodeProvisionResult struct {
	NodeID    uuid.UUID `json:"node_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role      string    `json:"role" binding:"required" example:"worker"`
	Labels    []string  `json:"labels" example:"zone=us-east,type=compute"`
	Succeeded bool      `json:"succeeded" example:"true"`
	Error     string    `json:"error" example:""`
}

// GCP Node Provision Request (with multiplier support)
type GCPNodeProvisionRequest struct {
	NamePrefix  string   `json:"name_prefix" binding:"required" example:"worker"`
	Number      int      `json:"number" binding:"required,min=1,max=20" example:"5"`
	Zone        string   `json:"zone" binding:"required" example:"us-central1-a"`
	MachineType string   `json:"machine_type" binding:"required" example:"n1-standard-2"`
	Role        string   `json:"role" binding:"required" example:"worker"`
	Labels      []string `json:"labels" example:"zone=us-central1"`
	DiskSizeGB  int      `json:"disk_size_gb" example:"100"`
	DiskType    string   `json:"disk_type" example:"pd-standard"`
}

// Provision Request Status
type ProvisionRequestStatus string

const (
	ProvisionStatusPending          ProvisionRequestStatus = "pending"
	ProvisionStatusPlanning         ProvisionRequestStatus = "planning"
	ProvisionStatusAwaitingApproval ProvisionRequestStatus = "awaiting_approval"
	ProvisionStatusApplying         ProvisionRequestStatus = "applying"
	ProvisionStatusCompleted        ProvisionRequestStatus = "completed"
	ProvisionStatusFailed           ProvisionRequestStatus = "failed"
	ProvisionStatusRejected         ProvisionRequestStatus = "rejected"
)

// Provision Request - tracks async node provisioning operations
type ProvisionRequest struct {
	ID         uuid.UUID              `json:"id" gorm:"type:uuid;primary_key"`
	Provider   string                 `json:"provider" gorm:"not null"` // gcp, aws, azure
	Status     ProvisionRequestStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	Request    datatypes.JSON         `json:"request" gorm:"type:jsonb;not null"` // Original request payload
	PlanOutput string                 `json:"plan_output" gorm:"type:text"`       // Terraform plan output
	NodeIDs    datatypes.JSON         `json:"node_ids" gorm:"type:jsonb"`         // Array of created node IDs
	Error      string                 `json:"error,omitempty" gorm:"type:text"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	DeletedAt  gorm.DeletedAt         `json:"-" gorm:"index"`
}

func (p *ProvisionRequest) BeforeCreate(tx *gorm.DB) error {
	if p.ID == (uuid.UUID{}) {
		p.ID = uuid.New()
	}
	return nil
}

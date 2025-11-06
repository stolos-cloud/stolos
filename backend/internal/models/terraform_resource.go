package models

import "time"

// TerraformResourceStatus represents the status of a resource operation
type TerraformResourceStatus string

const (
	ResourceStatusPending   TerraformResourceStatus = "pending"
	ResourceStatusCreating  TerraformResourceStatus = "creating"
	ResourceStatusModifying TerraformResourceStatus = "modifying"
	ResourceStatusDeleting  TerraformResourceStatus = "deleting"
	ResourceStatusComplete  TerraformResourceStatus = "complete"
	ResourceStatusFailed    TerraformResourceStatus = "failed"
	ResourceStatusSkipped   TerraformResourceStatus = "skipped"
)

// TerraformResourceUpdate represents a resource state update sent via WebSocket
type TerraformResourceUpdate struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Type        string                  `json:"type"`
	Provider    string                  `json:"provider"`
	Action      string                  `json:"action"`
	Status      TerraformResourceStatus `json:"status"`
	StartedAt   *time.Time              `json:"started_at,omitempty"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	Duration    string                  `json:"duration,omitempty"`
	Error       string                  `json:"error,omitempty"`
	Details     map[string]any          `json:"details,omitempty"`
	Parent      string                  `json:"parent,omitempty"`
}

// TerraformPlanResource represents a resource that will be affected in the plan
type TerraformPlanResource struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Provider string         `json:"provider"`
	Action   string         `json:"action"`
	Changes  map[string]any `json:"changes,omitempty"`
}

// TerraformWorkflowUpdate represents the overall workflow status
type TerraformWorkflowUpdate struct {
	Resources []TerraformResourceUpdate `json:"resources"`
	Summary   map[string]int            `json:"summary"`
	Outputs   map[string]any            `json:"outputs,omitempty"`
}

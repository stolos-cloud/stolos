package terraform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/stolos-cloud/stolos/backend/internal/models"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
)

// ResourceTracker handles parsing Terraform JSON output and sending resource updates
type ResourceTracker struct {
	session *wsservices.ApprovalSession
	resources map[string]*models.TerraformResourceUpdate
	workflow  *models.TerraformWorkflowUpdate
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker(session *wsservices.ApprovalSession) *ResourceTracker {
	return &ResourceTracker{
		session:   session,
		resources: make(map[string]*models.TerraformResourceUpdate),
		workflow: &models.TerraformWorkflowUpdate{
			Resources: []models.TerraformResourceUpdate{},
			Summary:   make(map[string]int),
		},
	}
}

// TerraformJSONMessage represents a single line of Terraform's JSON output
type TerraformJSONMessage struct {
	Type        string                 `json:"type"`
	Level       string                 `json:"@level"`
	Message     string                 `json:"@message"`
	Module      string                 `json:"@module,omitempty"`
	Timestamp   string                 `json:"@timestamp"`
	Hook        map[string]any `json:"hook,omitempty"`
	Diagnostic  *TerraformDiagnostic   `json:"diagnostic,omitempty"`
}

// TerraformDiagnostic represents diagnostic information
type TerraformDiagnostic struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail,omitempty"`
	Address  string `json:"address,omitempty"`
}

// InitializeWithPlan populates the resource tracker with planned resources
func (rt *ResourceTracker) InitializeWithPlan(plannedResources []models.TerraformResourceUpdate) {
	for _, resource := range plannedResources {
		resourceCopy := resource
		rt.resources[resource.ID] = &resourceCopy
	}
}

// StreamApplyJSON processes Terraform apply JSON output and sends resource updates
func (rt *ResourceTracker) StreamApplyJSON(ctx context.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg TerraformJSONMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if err := rt.processApplyMessage(msg); err != nil {
			log.Printf("Error processing apply message: %v", err)
		}
	}

	rt.sendWorkflowUpdate()

	return scanner.Err()
}


// processApplyMessage handles messages during the apply phase
func (rt *ResourceTracker) processApplyMessage(msg TerraformJSONMessage) error {
	switch msg.Type {
	case "apply_start":
		if msg.Hook != nil {
			if resourceData, ok := msg.Hook["resource"].(map[string]any); ok {
				addr := fmt.Sprintf("%v", resourceData["addr"])

				// Filter out data sources by checking address pattern
				// Data sources have addresses like "data.xxx" or "module.xxx.data.yyy"
				if strings.HasPrefix(addr, "data.") || strings.Contains(addr, ".data.") {
					return nil
				}

				resourceType := fmt.Sprintf("%v", resourceData["type"])
				rt.startResourceOperation(addr, resourceType)
			}
		}

	case "apply_complete":
		if msg.Hook != nil {
			if resourceData, ok := msg.Hook["resource"].(map[string]any); ok {
				addr := fmt.Sprintf("%v", resourceData["addr"])

				// Filter out data sources by checking address pattern
				if strings.HasPrefix(addr, "data.") || strings.Contains(addr, ".data.") {
					return nil
				}

				rt.completeResourceOperation(addr)
			}
		}

	case "apply_errored":
		if msg.Hook != nil {
			if resourceData, ok := msg.Hook["resource"].(map[string]any); ok {
				addr := fmt.Sprintf("%v", resourceData["addr"])
				rt.failResourceOperation(addr, msg.Diagnostic)
			}
		}

	case "diagnostic":
		if msg.Diagnostic != nil {
			severity := msg.Diagnostic.Severity
			if severity == "error" {
				rt.session.SendLog(fmt.Sprintf("Error: %s", msg.Diagnostic.Summary))
			}
		}

	case "outputs":
		if msg.Hook != nil {
			if outputs, ok := msg.Hook["outputs"].(map[string]any); ok {
				rt.workflow.Outputs = outputs
				rt.sendWorkflowUpdate()
			}
		}
	}

	return nil
}


// startResourceOperation marks a resource as being created/modified/deleted
func (rt *ResourceTracker) startResourceOperation(addr string, resourceType string) {
	now := time.Now()

	resource, exists := rt.resources[addr]
	if !exists {
		resource = &models.TerraformResourceUpdate{
			ID:        addr,
			Name:      extractResourceName(addr),
			Type:      resourceType,
			Provider:  extractProviderFromType(resourceType),
			Action:    "create",
		}
		rt.resources[addr] = resource
	}

	// Set status based on action
	switch resource.Action {
	case "delete":
		resource.Status = models.ResourceStatusDeleting
	case "update":
		resource.Status = models.ResourceStatusModifying
	default: // create or unknown
		resource.Status = models.ResourceStatusCreating
	}

	resource.StartedAt = &now

	rt.sendResourceUpdate(resource)
}

// completeResourceOperation marks a resource operation as complete
func (rt *ResourceTracker) completeResourceOperation(addr string) {
	resource, exists := rt.resources[addr]
	if !exists {
		return
	}

	now := time.Now()
	resource.CompletedAt = &now
	resource.Status = models.ResourceStatusComplete

	if resource.StartedAt != nil {
		duration := now.Sub(*resource.StartedAt)
		resource.Duration = duration.Round(time.Second).String()
	}

	rt.sendResourceUpdate(resource)
}

// failResourceOperation marks a resource operation as failed
func (rt *ResourceTracker) failResourceOperation(addr string, diagnostic *TerraformDiagnostic) {
	resource, exists := rt.resources[addr]
	if !exists {
		return
	}

	now := time.Now()
	resource.CompletedAt = &now
	resource.Status = models.ResourceStatusFailed

	if diagnostic != nil {
		resource.Error = diagnostic.Summary
		if diagnostic.Detail != "" {
			resource.Error = fmt.Sprintf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}

	if resource.StartedAt != nil {
		duration := now.Sub(*resource.StartedAt)
		resource.Duration = duration.Round(time.Second).String()
	}

	rt.sendResourceUpdate(resource)
}

// sendResourceUpdate sends a resource update via WebSocket
func (rt *ResourceTracker) sendResourceUpdate(resource *models.TerraformResourceUpdate) {
	if err := rt.session.SendResourceUpdate(*resource); err != nil {
		log.Printf("Failed to send resource update: %v", err)
	}
}

// sendWorkflowUpdate sends a workflow update via WebSocket
func (rt *ResourceTracker) sendWorkflowUpdate() {
	// Update resource list
	rt.workflow.Resources = make([]models.TerraformResourceUpdate, 0, len(rt.resources))
	for _, resource := range rt.resources {
		rt.workflow.Resources = append(rt.workflow.Resources, *resource)
	}

	if err := rt.session.SendWorkflowUpdate(*rt.workflow); err != nil {
		log.Printf("Failed to send workflow update: %v", err)
	}
}

// Helper functions

func extractResourceName(addr string) string {
	// Extract resource name from address like "module.node_xyz.google_compute_instance.node"
	parts := strings.Split(addr, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return addr
}
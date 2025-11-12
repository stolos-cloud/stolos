package terraform

import (
	"encoding/json"
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stolos-cloud/stolos/backend/internal/models"
)

// ParsePlanJSON parses the terraform plan JSON and extracts resource changes
func ParsePlanJSON(planJSON []byte) ([]models.TerraformResourceUpdate, error) {
	var tfPlan tfjson.Plan
	if err := json.Unmarshal(planJSON, &tfPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	var resources []models.TerraformResourceUpdate

	// Process resource changes
	for _, change := range tfPlan.ResourceChanges {
		if change == nil {
			continue
		}

		// Skip data sources
		if strings.HasPrefix(change.Address, "data.") || strings.Contains(change.Address, ".data.") {
			continue
		}

		// Determine action
		var action string

		if change.Change != nil && len(change.Change.Actions) > 0 {
			if len(change.Change.Actions) == 2 &&
				change.Change.Actions[0] == tfjson.ActionCreate &&
				change.Change.Actions[1] == tfjson.ActionDelete {
				action = "replace"
			} else {
				action = string(change.Change.Actions[0])
			}
		}

		// Skip no-op changes
		if action == "no-op" || action == "" {
			continue
		}

		resource := models.TerraformResourceUpdate{
			ID:       change.Address,
			Name:     change.Name,
			Type:     change.Type,
			Provider: extractProviderFromType(change.Type),
			Action:   action,
			Status:   models.ResourceStatusPending,
			Details:  extractResourceDetails(change),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// extractProviderFromType extracts provider from the resource type
func extractProviderFromType(resourceType string) string {
	// Extract provider from type like "google_compute_instance", "aws_instance", "azurerm_virtual_machine"
	parts := strings.Split(resourceType, "_")
	if len(parts) > 0 {
		return parts[0] // Returns: "google", "aws", "azurerm", etc.
	}
	return "unknown"
}

// extractResourceDetails extracts relevant details from the change
func extractResourceDetails(change *tfjson.ResourceChange) map[string]any {
	details := make(map[string]any)

	// Add basic info
	details["type"] = change.Type
	details["provider"] = change.ProviderName
	if change.ModuleAddress != "" {
		details["module"] = change.ModuleAddress
	}

	// Try to extract key values from After (create/update) or Before (delete)
	if change.Change != nil {
		var sourceMap map[string]any
		var ok bool

		// For delete operations, use Before; otherwise use After
		if change.Change.After != nil {
			sourceMap, ok = change.Change.After.(map[string]any)
		} else if change.Change.Before != nil {
			sourceMap, ok = change.Change.Before.(map[string]any)
		}

		if ok {
			// Extract commonly useful fields
			if name, ok := sourceMap["name"].(string); ok {
				details["name"] = name
			}
			if zone, ok := sourceMap["zone"].(string); ok {
				details["zone"] = zone
			}
			if region, ok := sourceMap["region"].(string); ok {
				details["region"] = region
			}
			if machineType, ok := sourceMap["machine_type"].(string); ok {
				details["machine_type"] = machineType
			}
			if size, ok := sourceMap["size"].(float64); ok {
				details["size"] = size
			}
		}
	}

	return details
}

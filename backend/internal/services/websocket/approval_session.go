package websocket

import (
	"context"
	"time"
)

// ApprovalSession extends BaseSession with approval workflow support
// Use this for workflows that require user approval
type ApprovalSession struct {
	*BaseSession
	approvalChan chan ApprovalResponse
}

// NewApprovalSession creates a new approval session
func NewApprovalSession(requestID string, client *Client) *ApprovalSession {
	return &ApprovalSession{
		BaseSession:  newBaseSession(requestID, client, SessionTypeApproval),
		approvalChan: make(chan ApprovalResponse, 1),
	}
}

// HandleMessage processes incoming approval messages
func (as *ApprovalSession) HandleMessage(msgType string, data map[string]any) error {
	// Handle approval actions
	if action, ok := data["action"].(string); ok {
		var response ApprovalResponse
		switch action {
		case "approve":
			response.Approved = true
			response.Message = "Approved by user"
		case "reject":
			response.Approved = false
			if reason, ok := data["reason"].(string); ok {
				response.Message = reason
			} else {
				response.Message = "Rejected by user"
			}
		default:
			// Unknown action, ignore
			return nil
		}

		// Send approval response (non-blocking)
		select {
		case as.approvalChan <- response:
		default:
			// Channel full, ignore
		}
	}
	return nil
}

// SendPlan sends terraform plan output
func (as *ApprovalSession) SendPlan(plan string) error {
	return as.client.SendPlan(plan)
}

// SendApprovalRequest sends an approval request to the client
func (as *ApprovalSession) SendApprovalRequest(summary string) error {
	return as.client.SendApprovalRequest(summary)
}

// WaitForApproval waits for user approval with timeout
func (as *ApprovalSession) WaitForApproval(timeout time.Duration) (bool, error) {
	select {
	case response := <-as.approvalChan:
		return response.Approved, nil
	case <-time.After(timeout):
		return false, context.DeadlineExceeded
	}
}

// WaitForApprovalCtx waits for user approval with context and timeout
func (as *ApprovalSession) WaitForApprovalCtx(ctx context.Context, timeout time.Duration) (bool, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case response := <-as.approvalChan:
		return response.Approved, nil
	case <-timer.C:
		return false, context.DeadlineExceeded
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// SendResourceUpdate sends a resource update to the client
func (as *ApprovalSession) SendResourceUpdate(resource any) error {
	return as.client.SendResourceUpdate(resource)
}

// SendWorkflowUpdate sends a workflow update to the client
func (as *ApprovalSession) SendWorkflowUpdate(workflow any) error {
	return as.client.SendWorkflowUpdate(workflow)
}

// Close cleans up approval session resources
func (as *ApprovalSession) Close() {
	close(as.approvalChan)
}

package websocket

import (
	"context"
	"time"
)

// Session represents a WebSocket session for a specific use case
type Session interface {
	// GetRequestID returns the unique identifier for this session
	GetRequestID() string

	// HandleMessage processes incoming messages from the client
	// Returns error if message handling fails
	HandleMessage(msgType string, data map[string]any) error

	// Close cleans up session resources
	Close()
}

// BaseSession provides a session that streams logs and status updates
// Use this for workflows that don't need incoming message handling
type BaseSession struct {
	requestID string
	client    *Client
}

// NewBaseSession creates a new base session
func NewBaseSession(requestID string, client *Client) *BaseSession {
	return &BaseSession{
		requestID: requestID,
		client:    client,
	}
}

// GetRequestID returns the request ID
func (bs *BaseSession) GetRequestID() string {
	return bs.requestID
}

// SendLog sends a log message
func (bs *BaseSession) SendLog(message string) error {
	return bs.client.SendLog(message)
}

// SendStatus sends a status update
func (bs *BaseSession) SendStatus(status string) error {
	return bs.client.SendStatus(status)
}

// SendError sends an error message
func (bs *BaseSession) SendError(err error) error {
	return bs.client.SendError(err.Error())
}

// SendErrorString sends an error message string
func (bs *BaseSession) SendErrorString(errMsg string) error {
	return bs.client.SendError(errMsg)
}

// SendComplete sends a completion message with optional data
func (bs *BaseSession) SendComplete(data any) error {
	return bs.client.SendComplete(data)
}

// HandleMessage default implementation - does nothing
func (bs *BaseSession) HandleMessage(msgType string, data map[string]any) error {
	// Base session doesn't handle incoming messages
	return nil
}

// Close default implementation - does nothing
func (bs *BaseSession) Close() {
	// No resources to clean up in base session
}

// ApprovalSession extends BaseSession with approval workflow support
// Use this for workflows that require user approval
type ApprovalSession struct {
	*BaseSession
	approvalChan chan ApprovalResponse
}

// NewApprovalSession creates a new approval session
func NewApprovalSession(requestID string, client *Client) *ApprovalSession {
	return &ApprovalSession{
		BaseSession:  NewBaseSession(requestID, client),
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

// Close cleans up approval session resources
func (as *ApprovalSession) Close() {
	close(as.approvalChan)
}

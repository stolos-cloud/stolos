package websocket

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

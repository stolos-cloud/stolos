package websocket

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types sent over WebSocket
const (
	MessageTypeLog      = "log"
	MessageTypeStatus   = "status"
	MessageTypePlan     = "plan"
	MessageTypeApproval = "approval_required"
	MessageTypeComplete = "complete"
	MessageTypeError    = "error"
)

// WebSocket message structure
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// ApprovalResponse represents a user's response to an approval request
type ApprovalResponse struct {
	Approved bool
	Message  string
}

// Client represents a WebSocket connection for a specific provision request
type Client struct {
	ID       string
	conn     *websocket.Conn
	send     chan Message
	approval chan ApprovalResponse // Channel for approval responses
	manager  *Manager
	mu       sync.Mutex
	isClosed bool
}

// Manager manages all active WebSocket connections
type Manager struct {
	clients    map[string]*Client // requestID -> Client
	register   chan *Client
	unregister chan *Client
	broadcast  map[string]chan Message // requestID -> message channel
	mu         sync.RWMutex
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(map[string]chan Message),
	}
}

// Run starts the manager's event loop
func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.ID] = client
			m.mu.Unlock()
			log.Printf("WebSocket client registered: %s", client.ID)

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client.ID]; ok {
				delete(m.clients, client.ID)
				close(client.send)
				log.Printf("WebSocket client unregistered: %s", client.ID)
			}
			m.mu.Unlock()
		}
	}
}

// RegisterClient registers a new WebSocket client for a provision request
func (m *Manager) RegisterClient(requestID string, conn *websocket.Conn) *Client {
	client := &Client{
		ID:       requestID,
		conn:     conn,
		send:     make(chan Message, 256),
		approval: make(chan ApprovalResponse, 1), // Buffered channel for approval
		manager:  m,
	}

	m.register <- client

	// Start reading and writing goroutines
	go client.writePump()
	go client.readPump()

	return client
}

// SendMessage sends a message to a specific provision request's WebSocket
func (m *Manager) SendMessage(requestID string, message Message) error {
	m.mu.RLock()
	client, ok := m.clients[requestID]
	m.mu.RUnlock()

	if !ok {
		return nil // Client not connected, skip silently
	}

	select {
	case client.send <- message:
		return nil
	default:
		// Channel full, close client
		m.unregister <- client
		return nil
	}
}

// writePump writes messages from the send channel to the WebSocket
func (c *Client) writePump() {
	defer func() {
		c.Close()
	}()

	for message := range c.send {
		c.mu.Lock()
		if c.isClosed {
			c.mu.Unlock()
			return
		}

		err := c.conn.WriteJSON(message)
		c.mu.Unlock()

		if err != nil {
			log.Printf("WebSocket write error for %s: %v", c.ID, err)
			return
		}
	}
}

// readPump reads messages from the WebSocket (for approval responses)
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.Close()
	}()

	for {
		var msg map[string]any
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for %s: %v", c.ID, err)
			}
			break
		}

		// Handle approval messages
		if action, ok := msg["action"].(string); ok {
			log.Printf("Received action from client %s: %s", c.ID, action)

			var response ApprovalResponse
			switch action {
			case "approve":
				response.Approved = true
				response.Message = "Approved by user"
			case "reject":
				response.Approved = false
				if reason, ok := msg["reason"].(string); ok {
					response.Message = reason
				} else {
					response.Message = "Rejected by user"
				}
			default:
				log.Printf("Unknown action: %s", action)
				continue
			}

			// Send approval response (non-blocking)
			select {
			case c.approval <- response:
				log.Printf("Approval response sent for %s: approved=%v", c.ID, response.Approved)
			default:
				log.Printf("Warning: approval channel full for %s", c.ID)
			}
		}
	}
}

// Close closes the WebSocket connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosed {
		c.isClosed = true
		c.conn.Close()
	}
}

// SendLog sends a log message to the client
func (c *Client) SendLog(log string) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypeLog,
		Payload: map[string]string{"message": log},
	})
}

// SendStatus sends a status update to the client
func (c *Client) SendStatus(status string) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypeStatus,
		Payload: map[string]string{"status": status},
	})
}

// SendPlan sends terraform plan output to the client
func (c *Client) SendPlan(plan string) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypePlan,
		Payload: map[string]string{"plan": plan},
	})
}

// SendApprovalRequest sends an approval request to the client
func (c *Client) SendApprovalRequest(summary string) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypeApproval,
		Payload: map[string]string{"summary": summary},
	})
}

// SendComplete sends a completion message with results
func (c *Client) SendComplete(data any) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypeComplete,
		Payload: data,
	})
}

// SendError sends an error message
func (c *Client) SendError(err string) error {
	return c.manager.SendMessage(c.ID, Message{
		Type:    MessageTypeError,
		Payload: map[string]string{"error": err},
	})
}

// WaitForApproval waits for user approval with timeout
func (c *Client) WaitForApproval(timeout time.Duration) (bool, error) {
	select {
	case response := <-c.approval:
		return response.Approved, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("approval timeout after %v", timeout)
	}
}

func (c *Client) WaitForApprovalCtx(ctx context.Context, timeout time.Duration) (bool, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case response := <-c.approval:
		return response.Approved, nil
	case <-timer.C:
		return false, fmt.Errorf("approval timeout after %v", timeout)
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

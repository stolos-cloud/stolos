package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
)

var eventStreamUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type EventHandlers struct {
	wsManager *wsservices.Manager
}

func NewEventHandlers(wsManager *wsservices.Manager) *EventHandlers {
	return &EventHandlers{
		wsManager: wsManager,
	}
}

// StreamEvents upgrades the HTTP connection and registers a websocket client
// that can receive platform-wide events.
func (h *EventHandlers) StreamEvents(c *gin.Context) {
	connectionID := c.Query("connection_id")
	if connectionID == "" {
		connectionID = uuid.NewString()
	}

	conn, err := eventStreamUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade to websocket"})
		return
	}

	client := h.wsManager.RegisterClient(connectionID, conn, nil)
	session := wsservices.NewEventSession(connectionID, client)
	_ = session.SendStatus("connected")
}

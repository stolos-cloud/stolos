package websocket

// EventSession is used for broadcasting platform events to connected clients.
type EventSession struct {
	*BaseSession
}

// NewEventSession creates a new event session.
func NewEventSession(requestID string, client *Client) *EventSession {
	return &EventSession{
		BaseSession: newBaseSession(requestID, client, SessionTypeEvent),
	}
}

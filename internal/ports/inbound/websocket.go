package inbound

import (
	"context"
)

// WebSocketHandler handles WebSocket connections for the gateway.
type WebSocketHandler interface {
	// HandleConnection handles a new WebSocket connection.
	// The connection is kept alive until closed by the client or server.
	HandleConnection(ctx context.Context, conn WebSocketConnection) error

	// Broadcast sends a message to all connected clients.
	Broadcast(ctx context.Context, msg *OutboundMessage) error
}

// WebSocketConnection represents an active WebSocket connection.
type WebSocketConnection interface {
	// Send sends a message to the client.
	Send(ctx context.Context, msg *OutboundMessage) error

	// Receive receives a message from the client.
	Receive(ctx context.Context) (*InboundMessage, error)

	// Close closes the connection.
	Close() error

	// UserID returns the user ID associated with this connection.
	UserID() string
}

package inbound

import (
	"context"
	"time"
)

// MessageHandler handles incoming messages from various messaging channels.
type MessageHandler interface {
	// Handle processes an incoming message and returns a response.
	// It should handle the full lifecycle of processing a message.
	Handle(ctx context.Context, msg *InboundMessage) (*OutboundMessage, error)
}

// InboundMessage represents an incoming message from a messaging channel.
type InboundMessage struct {
	ID        string
	UserID    string
	Channel   string
	Type      string
	Content   string
	Meta      map[string]any
	Timestamp time.Time
}

// OutboundMessage represents a response message to be sent back to a channel.
type OutboundMessage struct {
	To      string
	Channel string
	Type    string
	Content string
	Meta    map[string]any
}

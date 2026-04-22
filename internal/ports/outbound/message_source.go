package outbound

import (
	"context"
	"fmt"
)

// MessageSource is the interface for sending messages to various messaging channels.
type MessageSource interface {
	// Send sends a message to the specified channel.
	Send(ctx context.Context, channel string, to string, content string) error

	// SendWithMeta sends a message with metadata to the specified channel.
	SendWithMeta(ctx context.Context, channel string, to string, content string, meta map[string]any) error
}

// ErrChannelNotSupported is returned when a channel is not supported.
var ErrChannelNotSupported = fmt.Errorf("channel not supported")

// ErrSendFailed is returned when sending a message fails.
var ErrSendFailed = fmt.Errorf("failed to send message")

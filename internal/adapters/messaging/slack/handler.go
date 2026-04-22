// Package slack provides message handling for Slack bot.
package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/ports/inbound"
)

// Handler handles incoming Slack messages.
// It processes messages and generates responses using the message handler.
type Handler struct {
	messageHandler inbound.MessageHandler
	logger         zerolog.Logger
}

// NewHandler creates a new Slack message handler.
func NewHandler(messageHandler inbound.MessageHandler, logger zerolog.Logger) *Handler {
	return &Handler{
		messageHandler: messageHandler,
		logger:         logger,
	}
}

// Handle processes an incoming Slack message and returns a response.
// It converts the Slack message format to the inbound format,
// processes it through the message handler, and converts the response back.
func (h *Handler) Handle(ctx context.Context, msg *Message) (*OutboundMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("empty message")
	}

	// Convert to inbound message format
	inboundMsg := &inbound.InboundMessage{
		ID:        msg.Timestamp,
		UserID:    msg.User,
		Channel:   "slack",
		Type:      getMessageType(msg),
		Content:   msg.Text,
		Meta:      buildMessageMeta(msg),
		Timestamp: parseTimestamp(msg.Timestamp),
	}

	// Process through message handler
	response, err := h.messageHandler.Handle(ctx, inboundMsg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to handle message")
		return nil, fmt.Errorf("failed to handle message: %w", err)
	}

	// Convert response to Slack format
	return convertToSlackResponse(response), nil
}

// getMessageType determines the message type from the Slack message.
func getMessageType(msg *Message) string {
	if msg.Text != "" {
		return "text"
	}
	return "text"
}

// buildMessageMeta builds metadata from the Slack message.
func buildMessageMeta(msg *Message) map[string]any {
	return map[string]any{
		"channel":   msg.Channel,
		"timestamp": msg.Timestamp,
		"thread_ts": msg.ThreadTime,
		"user":      msg.User,
		"subtype":   msg.SubType,
		"bot_id":    msg.BotID,
	}
}

// parseTimestamp parses a Slack timestamp.
func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Now()
	}

	// Slack timestamps are Unix epoch in seconds with decimal
	// Try to parse as float
	var sec int64
	fmt.Sscanf(ts, "%d", &sec)

	return time.Unix(sec, 0)
}

// convertToSlackResponse converts an outbound message to Slack format.
func convertToSlackResponse(response *inbound.OutboundMessage) *OutboundMessage {
	if response == nil {
		return nil
	}

	return &OutboundMessage{
		To:      response.To,
		Channel: response.To,
		Type:    response.Type,
		Content: response.Content,
		Meta:    response.Meta,
	}
}

// OutboundMessage represents a response message for Slack.
type OutboundMessage struct {
	To      string
	Channel string
	Type    string
	Content string
	Meta    map[string]any
}

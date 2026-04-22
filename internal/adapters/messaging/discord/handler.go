// Package discord provides message handling for Discord bot.
package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/ports/inbound"
)

// Handler handles incoming Discord messages.
// It processes messages and generates responses using the message handler.
type Handler struct {
	messageHandler inbound.MessageHandler
	logger         zerolog.Logger
}

// NewHandler creates a new Discord message handler.
func NewHandler(messageHandler inbound.MessageHandler, logger zerolog.Logger) *Handler {
	return &Handler{
		messageHandler: messageHandler,
		logger:         logger,
	}
}

// Handle processes an incoming Discord message and returns a response.
// It converts the Discord message format to the inbound format,
// processes it through the message handler, and converts the response back.
func (h *Handler) Handle(ctx context.Context, msg *Message) (*OutboundMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("empty message")
	}

	// Convert to inbound message format
	inboundMsg := &inbound.InboundMessage{
		ID:        msg.ID,
		UserID:    msg.Author.ID,
		Channel:   "discord",
		Type:      getMessageType(msg),
		Content:   msg.Content,
		Meta:      buildMessageMeta(msg),
		Timestamp: parseTimestamp(msg.Timestamp),
	}

	// Process through message handler
	response, err := h.messageHandler.Handle(ctx, inboundMsg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to handle message")
		return nil, fmt.Errorf("failed to handle message: %w", err)
	}

	// Convert response to Discord format
	return convertToDiscordResponse(response), nil
}

// getMessageType determines the message type from the Discord message.
func getMessageType(msg *Message) string {
	if msg.Content != "" {
		return "text"
	}
	return "text"
}

// buildMessageMeta builds metadata from the Discord message.
func buildMessageMeta(msg *Message) map[string]any {
	return map[string]any{
		"channel_id":  msg.ChannelID,
		"message_id":  msg.ID,
		"author_id":   msg.Author.ID,
		"author_name": msg.Author.Username,
	}
}

// parseTimestamp parses a Discord timestamp.
func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Now()
	}
	// Discord uses ISO 8601-like format: 2024-01-01T00:00:00.000Z
	// Try to parse it
	t, err := time.Parse("2006-01-02T15:04:05.000Z", ts)
	if err != nil {
		return time.Now()
	}
	return t
}

// convertToDiscordResponse converts an outbound message to Discord format.
func convertToDiscordResponse(response *inbound.OutboundMessage) *OutboundMessage {
	if response == nil {
		return nil
	}

	return &OutboundMessage{
		To:      response.To,
		Channel: response.To, // channel ID
		Type:    response.Type,
		Content: response.Content,
		Meta:    response.Meta,
	}
}

// OutboundMessage represents a response message for Discord.
type OutboundMessage struct {
	To      string
	Channel string
	Type    string
	Content string
	Meta    map[string]any
}

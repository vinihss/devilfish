// Package telegram provides message handling for Telegram bot.
package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/ports/inbound"
)

// Handler handles incoming Telegram messages.
// It processes messages and generates responses using the message handler.
type Handler struct {
	messageHandler inbound.MessageHandler
	logger         zerolog.Logger
}

// NewHandler creates a new Telegram message handler.
func NewHandler(messageHandler inbound.MessageHandler, logger zerolog.Logger) *Handler {
	return &Handler{
		messageHandler: messageHandler,
		logger:         logger,
	}
}

// Handle processes an incoming Telegram update and returns a response.
// It converts the Telegram message format to the inbound format,
// processes it through the message handler, and converts the response back.
func (h *Handler) Handle(ctx context.Context, update *Update) (*OutboundMessage, error) {
	if update == nil || update.Message == nil {
		return nil, fmt.Errorf("empty update")
	}

	msg := update.Message

	// Convert to inbound message format
	inboundMsg := &inbound.InboundMessage{
		ID:        fmt.Sprintf("%d", msg.MessageID),
		UserID:    fmt.Sprintf("%d", msg.From.ID),
		Channel:   "telegram",
		Type:      getMessageType(msg),
		Content:   msg.Text,
		Meta:      buildMessageMeta(msg),
		Timestamp: time.Unix(int64(msg.Date), 0),
	}

	// Process through message handler
	response, err := h.messageHandler.Handle(ctx, inboundMsg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to handle message")
		return nil, fmt.Errorf("failed to handle message: %w", err)
	}

	// Convert response to Telegram format
	return convertToTelegramResponse(response, msg.Chat.ID), nil
}

// getMessageType determines the message type from the Telegram message.
func getMessageType(msg *Message) string {
	if msg.Text != "" {
		return "text"
	}
	return "text"
}

// buildMessageMeta builds metadata from the Telegram message.
func buildMessageMeta(msg *Message) map[string]any {
	return map[string]any{
		"chat_id":    fmt.Sprintf("%d", msg.Chat.ID),
		"message_id": msg.MessageID,
		"from_id":    fmt.Sprintf("%d", msg.From.ID),
		"from_name":  getUserName(msg.From),
	}
}

// getUserName returns the user's display name.
func getUserName(user *User) string {
	if user == nil {
		return ""
	}
	if user.Username != "" {
		return user.Username
	}
	if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}
	return ""
}

// convertToTelegramResponse converts an outbound message to Telegram format.
func convertToTelegramResponse(response *inbound.OutboundMessage, chatID int) *OutboundMessage {
	if response == nil {
		return nil
	}

	return &OutboundMessage{
		To:      fmt.Sprintf("%d", chatID),
		Channel: "telegram",
		Type:    response.Type,
		Content: response.Content,
		Meta:    response.Meta,
	}
}

// OutboundMessage represents a response message for Telegram.
type OutboundMessage struct {
	To      string
	Channel string
	Type    string
	Content string
	Meta    map[string]any
}

package usecase

import (
	"context"
	"fmt"

	"devilfish/internal/domain/valueobject"
	"devilfish/internal/infra/i18n"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/inbound"
)

// HandleMessageUseCase handles incoming messages from messaging channels.
// It processes messages and returns appropriate responses.
type HandleMessageUseCase struct {
	handler    inbound.MessageHandler
	logger     logging.Logger
	translator *i18n.Translator
}

// NewHandleMessageUseCase creates a new HandleMessageUseCase instance.
//
// Parameters:
//   - handler: The message handler for processing inbound messages
//   - logger: The logger instance
//   - translator: The translator for i18n support
//
// Returns a new HandleMessageUseCase instance.
func NewHandleMessageUseCase(
	handler inbound.MessageHandler,
	logger logging.Logger,
	translator *i18n.Translator,
) *HandleMessageUseCase {
	return &HandleMessageUseCase{
		handler:    handler,
		logger:     logger,
		translator: translator,
	}
}

// HandleInput represents the input for handling a message.
type HandleInput struct {
	Message *inbound.InboundMessage
}

// HandleOutput represents the output from handling a message.
type HandleOutput struct {
	Message   *inbound.OutboundMessage
	SessionID string
	Provider  *valueobject.Provider
}

// Handle processes an incoming message.
//
// Parameters:
//   - ctx: The context
//   - input: The input containing the message to handle
//
// Returns the handle output or an error.
func (uc *HandleMessageUseCase) Handle(ctx context.Context, input *HandleInput) (*HandleOutput, error) {
	if input == nil || input.Message == nil {
		return nil, fmt.Errorf("invalid input: message is required")
	}

	msg := input.Message
	uc.logger.With(map[string]interface{}{
		"message_id": msg.ID,
		"user_id":    msg.UserID,
		"channel":    msg.Channel,
		"type":       msg.Type,
	}).Info("handling message")

	// Process the message through the handler
	resp, err := uc.handler.Handle(ctx, msg)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"message_id": msg.ID,
			"error":      err.Error(),
		}).Error("failed to handle message")
		return nil, fmt.Errorf("failed to handle message %s: %w", msg.ID, err)
	}

	uc.logger.With(map[string]interface{}{
		"message_id":       msg.ID,
		"response_channel": resp.Channel,
	}).Info("message handled successfully")

	return &HandleOutput{
		Message:   resp,
		SessionID: msg.UserID,
	}, nil
}

// HandleWithSession processes an incoming message with existing session context.
//
// Parameters:
//   - ctx: The context
//   - input: The input containing the message
//   - sessionID: The session ID for context
//
// Returns the handle output or an error.
func (uc *HandleMessageUseCase) HandleWithSession(ctx context.Context, input *HandleInput, sessionID string) (*HandleOutput, error) {
	if input == nil || input.Message == nil {
		return nil, fmt.Errorf("invalid input: message is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("invalid input: session ID is required")
	}

	msg := input.Message
	uc.logger.With(map[string]interface{}{
		"message_id": msg.ID,
		"user_id":    msg.UserID,
		"session_id": sessionID,
	}).Info("handling message with session")

	// Process the message through the handler
	resp, err := uc.handler.Handle(ctx, msg)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"message_id": msg.ID,
			"session_id": sessionID,
			"error":      err.Error(),
		}).Error("failed to handle message with session")
		return nil, fmt.Errorf("failed to handle message %s with session %s: %w", msg.ID, sessionID, err)
	}

	uc.logger.With(map[string]interface{}{
		"message_id": msg.ID,
		"session_id": sessionID,
	}).Info("message handled successfully with session")

	return &HandleOutput{
		Message:   resp,
		SessionID: sessionID,
	}, nil
}

// GetLocalizedError returns a localized error message.
//
// Parameters:
//   - key: The translation key
//   - args: Optional arguments for formatting
//
// Returns the localized error message.
func (uc *HandleMessageUseCase) GetLocalizedError(key string, args ...interface{}) string {
	return uc.translator.Get(key, args...)
}

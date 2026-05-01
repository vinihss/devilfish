package messaging

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"devilfish/internal/application/usecase"
	"devilfish/internal/ports/inbound"
)

// sessionNamespace is a fixed UUID v1 namespace (DNS namespace per RFC 4122)
// used to derive deterministic session IDs from a user/channel pair.
var sessionNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// AIHandlerAdapter wraps ChatWithAIUseCase to implement inbound.MessageHandler.
// This allows chat interactions to work through the messaging adapter interface.
type AIHandlerAdapter struct {
	uc           *usecase.ChatWithAIUseCase
	systemPrompt string
}

// NewAIHandlerAdapter creates a new AIHandlerAdapter.
//
// Parameters:
//   - uc: The ChatWithAIUseCase to delegate chat requests to.
//   - systemPrompt: The system-level instruction that defines the agent's persona,
//     language, and behavior (e.g. "Você é um assistente em pt-BR...").
func NewAIHandlerAdapter(uc *usecase.ChatWithAIUseCase, systemPrompt string) *AIHandlerAdapter {
	return &AIHandlerAdapter{
		uc:           uc,
		systemPrompt: systemPrompt,
	}
}

// Handle implements the inbound.MessageHandler interface.
// It converts the inbound message to chat input and sends to the AI provider.
func (a *AIHandlerAdapter) Handle(ctx context.Context, msg *inbound.InboundMessage) (*inbound.OutboundMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is required")
	}

	// Derive a deterministic session ID from the user+channel pair so that
	// conversation history is preserved across messages from the same user.
	sessionID := deriveSessionID(msg.UserID, msg.Channel)

	// Get provider info for default model
	providerInfo := a.uc.GetProviderInfo()
	providerName := a.uc.GetProvider()

	// Build chat input from inbound message
	input := &usecase.ChatInput{
		Message:      msg.Content,
		SessionID:    &sessionID,
		UserID:       msg.UserID,
		Provider:     providerName,
		Model:        providerInfo.Model,
		SystemPrompt: a.systemPrompt,
		Temperature:  0.7,
		MaxTokens:    2048,
	}

	// Send to AI
	output, err := a.uc.Chat(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to process chat: %w", err)
	}

	// Build response
	response := &inbound.OutboundMessage{
		To:      msg.UserID,
		Channel: msg.Channel,
		Type:    "text",
		Content: output.Response.Content,
		Meta: map[string]any{
			"model":         output.Model,
			"provider":      output.Provider,
			"input_tokens":  output.InputToken,
			"output_tokens": output.OutputToken,
		},
	}

	return response, nil
}

// Compile-time check that AIHandlerAdapter implements inbound.MessageHandler.
var _ inbound.MessageHandler = (*AIHandlerAdapter)(nil)

// deriveSessionID produces a deterministic UUID for the given userID and channel
// combination. It encodes both components with their lengths to avoid collisions
// between inputs such as ("us", "er:ch") and ("user", ":ch").
func deriveSessionID(userID, channel string) uuid.UUID {
	// Length-prefix each component to prevent ambiguous concatenations.
	key := fmt.Sprintf("%d:%s|%d:%s", len(userID), userID, len(channel), channel)
	return uuid.NewSHA1(sessionNamespace, []byte(key))
}

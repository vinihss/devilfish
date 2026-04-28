package messaging

import (
	"context"
	"fmt"

	"devilfish/internal/application/usecase"
	"devilfish/internal/ports/inbound"
)

// AIHandlerAdapter wraps ChatWithAIUseCase to implement inbound.MessageHandler.
// This allows chat interactions to work through the messaging adapter interface.
type AIHandlerAdapter struct {
	uc           *usecase.ChatWithAIUseCase
	systemPrompt string
}

// NewAIHandlerAdapter creates a new AIHandlerAdapter.
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

	// Get provider info for default model
	providerInfo := a.uc.GetProviderInfo()
	providerName := a.uc.GetProvider()

	// Build chat input from inbound message
	input := &usecase.ChatInput{
		Message:      msg.Content,
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

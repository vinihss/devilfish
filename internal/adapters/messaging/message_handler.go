package messaging

import (
	"context"

	"devilfish/internal/application/usecase"
	"devilfish/internal/ports/inbound"
)

// MessageHandlerAdapter wraps HandleMessageUseCase to implement inbound.MessageHandler.
// This allows the use case to be used directly with messaging adapters.
type MessageHandlerAdapter struct {
	uc *usecase.HandleMessageUseCase
}

// NewMessageHandlerAdapter creates a new MessageHandlerAdapter.
func NewMessageHandlerAdapter(uc *usecase.HandleMessageUseCase) *MessageHandlerAdapter {
	return &MessageHandlerAdapter{
		uc: uc,
	}
}

// Handle implements the inbound.MessageHandler interface.
func (a *MessageHandlerAdapter) Handle(ctx context.Context, msg *inbound.InboundMessage) (*inbound.OutboundMessage, error) {
	input := &usecase.HandleInput{
		Message: msg,
	}

	output, err := a.uc.Handle(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.Message, nil
}

// Compile-time check that MessageHandlerAdapter implements inbound.MessageHandler.
var _ inbound.MessageHandler = (*MessageHandlerAdapter)(nil)

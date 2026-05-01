package usecase

import (
	"context"
	"fmt"
	"time"

	"devilfish/internal/domain/valueobject"
	"devilfish/internal/infra/i18n"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
	"github.com/google/uuid"
)

// ChatWithAIUseCase handles chat interactions with AI providers.
// It manages the conversation flow between users and AI services.
type ChatWithAIUseCase struct {
	provider          outbound.AIProvider
	store             outbound.SessionStore
	logger            logging.Logger
	translator        *i18n.Translator
	model             string // Default model from config
	assembler         *ContextAssembler
	embeddingProvider outbound.EmbeddingProvider
	embeddingStore    outbound.EmbeddingStore
}

// NewChatWithAIUseCase creates a new ChatWithAIUseCase instance.
//
// Parameters:
//   - provider: The AI provider for chat interactions
//   - store: The session store for persisting conversation context
//   - logger: The logger instance
//   - translator: The translator for i18n support
//   - model: The default model name
//
// Returns a new ChatWithAIUseCase instance.
func NewChatWithAIUseCase(
	provider outbound.AIProvider,
	store outbound.SessionStore,
	logger logging.Logger,
	translator *i18n.Translator,
	model string,
) *ChatWithAIUseCase {
	return &ChatWithAIUseCase{
		provider:   provider,
		store:      store,
		logger:     logger,
		translator: translator,
		model:      model,
		assembler:  NewContextAssembler(DefaultContextAssemblerConfig()),
	}
}

// WithEmbeddings enables hybrid context retrieval (recent + semantic memory).
//
// When both an EmbeddingProvider and EmbeddingStore are supplied, each user
// message is embedded on persistence and semantic search is used to enrich the
// context window sent to the LLM.
func (uc *ChatWithAIUseCase) WithEmbeddings(ep outbound.EmbeddingProvider, es outbound.EmbeddingStore) *ChatWithAIUseCase {
	uc.embeddingProvider = ep
	uc.embeddingStore = es
	return uc
}

// WithContextAssembler replaces the default ContextAssembler.
func (uc *ChatWithAIUseCase) WithContextAssembler(ca *ContextAssembler) *ChatWithAIUseCase {
	uc.assembler = ca
	return uc
}

// ChatInput represents the input for a chat request.
type ChatInput struct {
	Message      string     // User's message content
	SessionID    *uuid.UUID // Optional session ID
	UserID       string     // User identifier
	Provider     string     // AI provider name (e.g., "openai", "groq")
	Model        string     // Model name (e.g., "gpt-4", "llama-3")
	SystemPrompt string     // Optional system prompt
	Temperature  float64    // Temperature setting (0.0-2.0)
	MaxTokens    int        // Maximum tokens in response
}

// ChatOutput represents the output from a chat request.
type ChatOutput struct {
	Response    *outbound.ChatResponse
	SessionID   uuid.UUID
	Provider    string
	Model       string
	InputToken  int
	OutputToken int
}

// Chat sends a chat message to the AI provider and returns the response.
//
// Parameters:
//   - ctx: The context
//   - input: The chat input containing the message and settings
//
// Returns the chat output with AI response or an error.
func (uc *ChatWithAIUseCase) Chat(ctx context.Context, input *ChatInput) (*ChatOutput, error) {
	if err := uc.validateChatInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"user_id":        input.UserID,
		"provider":       input.Provider,
		"model":          input.Model,
		"message_length": len(input.Message),
	}).Info("processing chat request")

	// Check provider availability
	if !uc.provider.IsAvailable() {
		errMsg := uc.translator.Get("error.provider.unavailable")
		uc.logger.With(map[string]interface{}{
			"provider": input.Provider,
		}).Error("AI provider unavailable")
		return nil, fmt.Errorf("%s: %s", errMsg, input.Provider)
	}

	// Get or create session
	sessionID := input.SessionID
	if sessionID == nil {
		newID := uuid.New()
		sessionID = &newID
	}

	session, err := uc.store.Get(ctx, *sessionID)
	if err != nil {
		// Create new session if not found
		session = &outbound.Session{
			ID:        *sessionID,
			UserID:    input.UserID,
			Provider:  input.Provider,
			Channel:   input.Provider,
			Context:   make([]outbound.Message, 0),
			Meta:      make(map[string]any),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Retrieve semantically relevant memory when hybrid retrieval is enabled.
	var relevantMemory []*outbound.StoredEmbedding
	if uc.embeddingProvider != nil && uc.embeddingStore != nil {
		vec, embedErr := uc.embeddingProvider.GenerateEmbedding(ctx, input.Message)
		if embedErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      embedErr.Error(),
			}).Warn("failed to generate embedding for retrieval; skipping vector search")
		} else {
			var searchErr error
			relevantMemory, searchErr = uc.embeddingStore.SearchSimilar(ctx, sessionID.String(), vec, 5, "")
			if searchErr != nil {
				uc.logger.With(map[string]interface{}{
					"session_id": sessionID,
					"error":      searchErr.Error(),
				}).Warn("failed to search similar embeddings; skipping vector memory")
			}
		}
	}

	// Build messages using the context assembler.
	messages := uc.assembler.Build(AssembleInput{
		SystemPrompt:   input.SystemPrompt,
		RecentMessages: session.Context,
		RelevantMemory: relevantMemory,
		UserInput:      input.Message,
	})

	// Create chat request
	req := &outbound.ChatRequest{
		Model:        input.Model,
		Messages:     messages,
		SystemPrompt: "", // already embedded by the assembler
		Temperature:  input.Temperature,
		MaxTokens:    input.MaxTokens,
	}

	// Send to AI provider
	resp, err := uc.provider.Chat(ctx, req)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		}).Error("chat request failed")
		return nil, fmt.Errorf("chat request failed: %w", err)
	}

	// Persist user message in session timeline.
	userMsg := outbound.Message{
		Role:    "user",
		Content: input.Message,
		Time:    time.Now(),
	}
	session.Context = append(session.Context, userMsg)

	// Persist assistant response.
	session.Context = append(session.Context, outbound.Message{
		Role:    "assistant",
		Content: resp.Content,
		Time:    time.Now(),
	})
	session.UpdatedAt = time.Now()

	// Save session
	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		}).Warn("failed to save session")
		// Continue anyway - don't fail the chat
	}

	// Persist embedding for the user message when hybrid retrieval is enabled.
	if uc.embeddingProvider != nil && uc.embeddingStore != nil {
		msgID := uuid.New().String()
		vec, embedErr := uc.embeddingProvider.GenerateEmbedding(ctx, input.Message)
		if embedErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      embedErr.Error(),
			}).Warn("failed to generate embedding for persistence; skipping")
		} else if saveErr := uc.embeddingStore.Save(ctx, &outbound.StoredEmbedding{
			MessageID: msgID,
			SessionID: sessionID.String(),
			Vector:    vec,
			Content:   input.Message,
		}); saveErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      saveErr.Error(),
			}).Warn("failed to persist embedding; skipping")
		}
	}

	var inputTokens, outputTokens int
	if resp.Usage != nil {
		inputTokens = resp.Usage.InputTokens
		outputTokens = resp.Usage.OutputTokens
	}

	uc.logger.With(map[string]interface{}{
		"session_id":    sessionID,
		"provider":      input.Provider,
		"model":         input.Model,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
	}).Info("chat request completed")

	return &ChatOutput{
		Response:    resp,
		SessionID:   *sessionID,
		Provider:    input.Provider,
		Model:       input.Model,
		InputToken:  inputTokens,
		OutputToken: outputTokens,
	}, nil
}

// StreamChat sends a streaming chat message to the AI provider.
//
// Parameters:
//   - ctx: The context
//   - input: The chat input
//   - onChunk: Callback for each response chunk
//
// Returns an error if streaming fails.
func (uc *ChatWithAIUseCase) StreamChat(ctx context.Context, input *ChatInput, onChunk func(string)) error {
	if err := uc.validateChatInput(input); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}
	if onChunk == nil {
		return fmt.Errorf("onChunk callback is required")
	}

	uc.logger.With(map[string]interface{}{
		"user_id":  input.UserID,
		"provider": input.Provider,
		"model":    input.Model,
	}).Info("processing streaming chat request")

	// Check provider availability
	if !uc.provider.IsAvailable() {
		errMsg := uc.translator.Get("error.provider.unavailable")
		uc.logger.With(map[string]interface{}{
			"provider": input.Provider,
		}).Error("AI provider unavailable")
		return fmt.Errorf("%s: %s", errMsg, input.Provider)
	}

	// Get or create session
	sessionID := input.SessionID
	if sessionID == nil {
		newID := uuid.New()
		sessionID = &newID
	}

	session, err := uc.store.Get(ctx, *sessionID)
	if err != nil || session == nil {
		session = &outbound.Session{
			ID:        *sessionID,
			UserID:    input.UserID,
			Provider:  input.Provider,
			Channel:   input.Provider,
			Context:   make([]outbound.Message, 0),
			Meta:      make(map[string]any),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Retrieve semantically relevant memory when hybrid retrieval is enabled.
	var relevantMemory []*outbound.StoredEmbedding
	if uc.embeddingProvider != nil && uc.embeddingStore != nil {
		vec, embedErr := uc.embeddingProvider.GenerateEmbedding(ctx, input.Message)
		if embedErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      embedErr.Error(),
			}).Warn("failed to generate embedding for retrieval; skipping vector search")
		} else {
			relevantMemory, _ = uc.embeddingStore.SearchSimilar(ctx, sessionID.String(), vec, 5, "")
		}
	}

	// Build messages using the context assembler.
	messages := uc.assembler.Build(AssembleInput{
		SystemPrompt:   input.SystemPrompt,
		RecentMessages: session.Context,
		RelevantMemory: relevantMemory,
		UserInput:      input.Message,
	})

	// Create chat request
	req := &outbound.ChatRequest{
		Model:        input.Model,
		Messages:     messages,
		SystemPrompt: "", // already embedded by the assembler
		Temperature:  input.Temperature,
		MaxTokens:    input.MaxTokens,
	}

	// Stream from AI provider
	err = uc.provider.StreamChat(ctx, req, onChunk)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		}).Error("streaming chat request failed")
		return fmt.Errorf("streaming chat request failed: %w", err)
	}

	// Persist user message in session timeline.
	session.Context = append(session.Context, outbound.Message{
		Role:    "user",
		Content: input.Message,
		Time:    time.Now(),
	})
	session.UpdatedAt = time.Now()

	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		}).Warn("failed to save session after streaming")
	}

	// Persist embedding for the user message when hybrid retrieval is enabled.
	if uc.embeddingProvider != nil && uc.embeddingStore != nil {
		msgID := uuid.New().String()
		vec, embedErr := uc.embeddingProvider.GenerateEmbedding(ctx, input.Message)
		if embedErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      embedErr.Error(),
			}).Warn("failed to generate embedding for persistence; skipping")
		} else if saveErr := uc.embeddingStore.Save(ctx, &outbound.StoredEmbedding{
			MessageID: msgID,
			SessionID: sessionID.String(),
			Vector:    vec,
			Content:   input.Message,
		}); saveErr != nil {
			uc.logger.With(map[string]interface{}{
				"session_id": sessionID,
				"error":      saveErr.Error(),
			}).Warn("failed to persist embedding; skipping")
		}
	}

	uc.logger.With(map[string]interface{}{
		"session_id": sessionID,
	}).Info("streaming chat request completed")

	return nil
}

// validateChatInput validates the chat input.
func (uc *ChatWithAIUseCase) validateChatInput(input *ChatInput) error {
	if input == nil {
		return fmt.Errorf("input is required")
	}
	if input.Message == "" {
		return fmt.Errorf("message is required")
	}
	if input.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if input.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	// Model is optional - use config default or skip
	if input.Model == "" {
		input.Model = uc.model // Use default from use case
	}
	if input.Temperature < 0 || input.Temperature > 2.0 {
		input.Temperature = 0.7 // Default temperature
	}
	if input.MaxTokens <= 0 {
		input.MaxTokens = 2048 // Default max tokens
	}
	return nil
}

// GetProvider returns the AI provider name.
//
// Returns the provider name.
func (uc *ChatWithAIUseCase) GetProvider() string {
	return uc.provider.Name()
}

// GetAvailableModels returns available models for the current provider.
//
// Returns a slice of available model names.
func (uc *ChatWithAIUseCase) GetAvailableModels() []string {
	// This could be extended to fetch from provider
	return []string{}
}

// GetProviderInfo returns information about the current provider.
//
// Returns provider information as a valueobject.Provider.
func (uc *ChatWithAIUseCase) GetProviderInfo() *valueobject.Provider {
	return &valueobject.Provider{
		Name:     uc.provider.Name(),
		Model:    uc.model,
		Endpoint: "",
	}
}

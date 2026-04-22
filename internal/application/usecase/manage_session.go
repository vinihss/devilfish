package usecase

import (
	"context"
	"fmt"
	"time"

	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
	"github.com/google/uuid"
)

// ManageSessionUseCase manages user sessions for chat interactions.
// It handles session CRUD operations and context management.
type ManageSessionUseCase struct {
	store  outbound.SessionStore
	logger logging.Logger
}

// NewManageSessionUseCase creates a new ManageSessionUseCase instance.
//
// Parameters:
//   - store: The session store for persistence
//   - logger: The logger instance
//
// Returns a new ManageSessionUseCase instance.
func NewManageSessionUseCase(
	store outbound.SessionStore,
	logger logging.Logger,
) *ManageSessionUseCase {
	return &ManageSessionUseCase{
		store:  store,
		logger: logger,
	}
}

// CreateInput represents the input for creating a session.
type CreateInput struct {
	UserID   string         // User identifier
	Provider string         // AI provider name
	Channel  string         // Channel name
	Meta     map[string]any // Optional metadata
}

// CreateOutput represents the output from creating a session.
type CreateOutput struct {
	Session *outbound.Session
}

// Create creates a new session.
//
// Parameters:
//   - ctx: The context
//   - input: The create input
//
// Returns the create output with the new session or an error.
func (uc *ManageSessionUseCase) Create(ctx context.Context, input *CreateInput) (*CreateOutput, error) {
	if err := uc.validateCreateInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	sessionID := uuid.New()
	now := time.Now()

	session := &outbound.Session{
		ID:        sessionID,
		UserID:    input.UserID,
		Provider:  input.Provider,
		Channel:   input.Channel,
		Context:   make([]outbound.Message, 0),
		Meta:      input.Meta,
		CreatedAt: now,
		UpdatedAt: now,
	}

	uc.logger.With(map[string]interface{}{
		"session_id": sessionID,
		"user_id":    input.UserID,
		"provider":   input.Provider,
		"channel":    input.Channel,
	}).Info("creating session")

	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		}).Error("failed to create session")
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"session_id": sessionID,
	}).Info("session created successfully")

	return &CreateOutput{
		Session: session,
	}, nil
}

// GetInput represents the input for getting a session.
type GetInput struct {
	SessionID uuid.UUID
}

// GetOutput represents the output from getting a session.
type GetOutput struct {
	Session *outbound.Session
}

// Get retrieves a session by ID.
//
// Parameters:
//   - ctx: The context
//   - input: The get input
//
// Returns the get output with the session or an error.
func (uc *ManageSessionUseCase) Get(ctx context.Context, input *GetInput) (*GetOutput, error) {
	if input == nil || input.SessionID == uuid.Nil {
		return nil, fmt.Errorf("invalid input: session ID is required")
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Debug("retrieving session")

	session, err := uc.store.Get(ctx, input.SessionID)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("session not found")
		return nil, fmt.Errorf("session not found: %w", err)
	}

	return &GetOutput{
		Session: session,
	}, nil
}

// UpdateInput represents the input for updating a session.
type UpdateInput struct {
	SessionID uuid.UUID
	UserID    string
	Provider  string
	Channel   string
	Meta      map[string]any
}

// Update updates an existing session.
//
// Parameters:
//   - ctx: The context
//   - input: The update input
//
// Returns an error if the update fails.
func (uc *ManageSessionUseCase) Update(ctx context.Context, input *UpdateInput) error {
	if input == nil || input.SessionID == uuid.Nil {
		return fmt.Errorf("invalid input: session ID is required")
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("updating session")

	session, err := uc.store.Get(ctx, input.SessionID)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("session not found for update")
		return fmt.Errorf("session not found: %w", err)
	}

	// Update fields if provided
	if input.UserID != "" {
		session.UserID = input.UserID
	}
	if input.Provider != "" {
		session.Provider = input.Provider
	}
	if input.Channel != "" {
		session.Channel = input.Channel
	}
	if input.Meta != nil {
		session.Meta = input.Meta
	}
	session.UpdatedAt = time.Now()

	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("failed to update session")
		return fmt.Errorf("failed to update session: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("session updated successfully")

	return nil
}

// DeleteInput represents the input for deleting a session.
type DeleteInput struct {
	SessionID uuid.UUID
}

// Delete deletes a session by ID.
//
// Parameters:
//   - ctx: The context
//   - input: The delete input
//
// Returns an error if the delete fails.
func (uc *ManageSessionUseCase) Delete(ctx context.Context, input *DeleteInput) error {
	if input == nil || input.SessionID == uuid.Nil {
		return fmt.Errorf("invalid input: session ID is required")
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("deleting session")

	if err := uc.store.Delete(ctx, input.SessionID); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("session deleted successfully")

	return nil
}

// ListByUserInput represents the input for listing sessions by user.
type ListByUserInput struct {
	UserID string
}

// ListByUserOutput represents the output from listing sessions by user.
type ListByUserOutput struct {
	Sessions []*outbound.Session
	Count    int
}

// ListByUser lists all sessions for a user.
//
// Parameters:
//   - ctx: The context
//   - input: The list input
//
// Returns the list output with sessions or an error.
func (uc *ManageSessionUseCase) ListByUser(ctx context.Context, input *ListByUserInput) (*ListByUserOutput, error) {
	if input == nil || input.UserID == "" {
		return nil, fmt.Errorf("invalid input: user ID is required")
	}

	uc.logger.With(map[string]interface{}{
		"user_id": input.UserID,
	}).Debug("listing sessions for user")

	sessions, err := uc.store.ListByUser(ctx, input.UserID)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"user_id": input.UserID,
			"error":   err.Error(),
		}).Error("failed to list sessions")
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"user_id": input.UserID,
		"count":   len(sessions),
	}).Debug("sessions listed")

	return &ListByUserOutput{
		Sessions: sessions,
		Count:    len(sessions),
	}, nil
}

// AddMessageInput represents the input for adding a message to a session.
type AddMessageInput struct {
	SessionID uuid.UUID
	Role      string // "user" or "assistant"
	Content   string
}

// AddMessage adds a message to a session's context.
//
// Parameters:
//   - ctx: The context
//   - input: The add message input
//
// Returns an error if the operation fails.
func (uc *ManageSessionUseCase) AddMessage(ctx context.Context, input *AddMessageInput) error {
	if input == nil || input.SessionID == uuid.Nil {
		return fmt.Errorf("invalid input: session ID is required")
	}
	if input.Role == "" {
		return fmt.Errorf("invalid input: role is required")
	}
	if input.Content == "" {
		return fmt.Errorf("invalid input: content is required")
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
		"role":       input.Role,
	}).Debug("adding message to session")

	session, err := uc.store.Get(ctx, input.SessionID)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("session not found for adding message")
		return fmt.Errorf("session not found: %w", err)
	}

	msg := outbound.Message{
		Role:    input.Role,
		Content: input.Content,
		Time:    time.Now(),
	}

	session.Context = append(session.Context, msg)
	session.UpdatedAt = time.Now()

	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("failed to save session after adding message")
		return fmt.Errorf("failed to save session: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Debug("message added to session")

	return nil
}

// ClearMessagesInput represents the input for clearing session messages.
type ClearMessagesInput struct {
	SessionID uuid.UUID
}

// ClearMessages clears all messages from a session's context.
//
// Parameters:
//   - ctx: The context
//   - input: The clear messages input
//
// Returns an error if the operation fails.
func (uc *ManageSessionUseCase) ClearMessages(ctx context.Context, input *ClearMessagesInput) error {
	if input == nil || input.SessionID == uuid.Nil {
		return fmt.Errorf("invalid input: session ID is required")
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("clearing messages from session")

	session, err := uc.store.Get(ctx, input.SessionID)
	if err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("session not found for clearing messages")
		return fmt.Errorf("session not found: %w", err)
	}

	session.Context = make([]outbound.Message, 0)
	session.UpdatedAt = time.Now()

	if err := uc.store.Save(ctx, session); err != nil {
		uc.logger.With(map[string]interface{}{
			"session_id": input.SessionID,
			"error":      err.Error(),
		}).Error("failed to save session after clearing messages")
		return fmt.Errorf("failed to save session: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"session_id": input.SessionID,
	}).Info("messages cleared from session")

	return nil
}

// validateCreateInput validates the create input.
func (uc *ManageSessionUseCase) validateCreateInput(input *CreateInput) error {
	if input == nil {
		return fmt.Errorf("input is required")
	}
	if input.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if input.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if input.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	return nil
}

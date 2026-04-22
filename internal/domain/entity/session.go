package entity

import (
	"errors"
	"time"
)

// SessionContext holds the context data for a session.
type SessionContext struct {
	ProviderName string            `json:"provider_name"` // AI provider name
	Model        string            `json:"model"`         // Model being used
	SystemPrompt string            `json:"system_prompt"` // System prompt for context
	Variables    map[string]string `json:"variables"`     // Session variables
}

// ErrInvalidSession is returned when session data is invalid.
var ErrInvalidSession = errors.New("invalid session")

// Session represents a user session entity.
type Session struct {
	ID        string          `json:"id"`                 // Unique identifier
	UserID    string          `json:"user_id"`            // User who owns the session
	Messages  []*Message      `json:"messages"`           // Message history
	Context   *SessionContext `json:"context"`            // Session context
	CreatedAt time.Time       `json:"created_at"`         // When session was created
	UpdatedAt time.Time       `json:"updated_at"`         // Last time session was updated
	IsActive  bool            `json:"is_active"`          // Whether session is active
	Provider  *string         `json:"provider,omitempty"` // Current AI provider
}

// NewSession creates a new Session entity.
func NewSession(id, userID string) (*Session, error) {
	if id == "" {
		return nil, errors.New("session id is required")
	}
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	now := time.Now()
	return &Session{
		ID:        id,
		UserID:    userID,
		Messages:  make([]*Message, 0),
		Context:   newDefaultSessionContext(),
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}, nil
}

// newDefaultSessionContext creates default session context.
func newDefaultSessionContext() *SessionContext {
	return &SessionContext{
		Variables: make(map[string]string),
	}
}

// AddMessage adds a message to the session history.
func (s *Session) AddMessage(msg *Message) error {
	if msg == nil {
		return ErrInvalidSession
	}
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
	return nil
}

// ClearHistory removes all messages from the session.
func (s *Session) ClearHistory() {
	s.Messages = make([]*Message, 0)
	s.UpdatedAt = time.Now()
}

// MessageCount returns the number of messages in the session.
func (s *Session) MessageCount() int {
	return len(s.Messages)
}

// SetContext updates the session context.
func (s *Session) SetContext(ctx *SessionContext) error {
	if ctx == nil {
		return ErrInvalidSession
	}
	s.Context = ctx
	s.UpdatedAt = time.Now()
	return nil
}

// SetProvider sets the AI provider for the session.
func (s *Session) SetProvider(provider string) {
	s.Provider = &provider
	s.UpdatedAt = time.Now()
}

// Deactivate marks the session as inactive.
func (s *Session) Deactivate() {
	s.IsActive = false
	s.UpdatedAt = time.Now()
}

// Activate marks the session as active.
func (s *Session) Activate() {
	s.IsActive = true
	s.UpdatedAt = time.Now()
}

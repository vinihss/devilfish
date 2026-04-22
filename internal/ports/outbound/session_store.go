package outbound

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionStore is the interface for storing and retrieving user sessions.
type SessionStore interface {
	// Get retrieves a session by ID.
	Get(ctx context.Context, sessionID uuid.UUID) (*Session, error)

	// Save saves a session.
	Save(ctx context.Context, session *Session) error

	// Delete deletes a session by ID.
	Delete(ctx context.Context, sessionID uuid.UUID) error

	// ListByUser lists all sessions for a user.
	ListByUser(ctx context.Context, userID string) ([]*Session, error)
}

// Session represents a user session.
type Session struct {
	ID        uuid.UUID
	UserID    string
	Provider  string
	Channel   string
	Context   []Message
	Meta      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message represents a message in a session context.
type Message struct {
	Role    string
	Content string
	Time    time.Time
}

// ErrSessionNotFound is returned when a session is not found.
func ErrSessionNotFound(sessionID uuid.UUID) error {
	return fmt.Errorf("session not found: %s", sessionID)
}

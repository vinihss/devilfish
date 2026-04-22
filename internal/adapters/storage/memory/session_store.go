// Package memory provides an in-memory implementation of the SessionStore.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"devilfish/internal/ports/outbound"
)

// SessionStore is an in-memory implementation of the SessionStore interface.
// It uses sync.RWMutex for thread-safe concurrent access.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*outbound.Session
	userIdx  map[string]map[uuid.UUID]struct{} // userID -> set of sessionIDs
}

// NewSessionStore creates a new in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[uuid.UUID]*outbound.Session),
		userIdx:  make(map[string]map[uuid.UUID]struct{}),
	}
}

// Get retrieves a session by ID.
//
// Returns outbound.ErrSessionNotFound if the session does not exist.
func (s *SessionStore) Get(ctx context.Context, sessionID uuid.UUID) (*outbound.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, outbound.ErrSessionNotFound(sessionID)
	}

	return session, nil
}

// Save saves a session. If the session already exists, it will be updated.
//
// The session's UpdatedAt field is automatically set to the current time.
func (s *SessionStore) Save(ctx context.Context, session *outbound.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// Check if this is a new session or update
	isNew := true
	if _, exists := s.sessions[session.ID]; exists {
		isNew = false
	}

	// Update timestamps
	session.UpdatedAt = now
	if isNew {
		session.CreatedAt = now
	}

	// Store the session
	s.sessions[session.ID] = session

	// Update user index
	if isNew {
		if s.userIdx[session.UserID] == nil {
			s.userIdx[session.UserID] = make(map[uuid.UUID]struct{})
		}
		s.userIdx[session.UserID][session.ID] = struct{}{}
	}

	return nil
}

// Delete deletes a session by ID.
//
// Returns outbound.ErrSessionNotFound if the session does not exist.
func (s *SessionStore) Delete(ctx context.Context, sessionID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return outbound.ErrSessionNotFound(sessionID)
	}

	// Remove from main storage
	delete(s.sessions, sessionID)

	// Remove from user index
	if userSessions, exists := s.userIdx[session.UserID]; exists {
		delete(userSessions, sessionID)
		if len(userSessions) == 0 {
			delete(s.userIdx, session.UserID)
		}
	}

	return nil
}

// ListByUser lists all sessions for a user.
//
// Returns an empty slice if the user has no sessions.
func (s *SessionStore) ListByUser(ctx context.Context, userID string) ([]*outbound.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userSessions, ok := s.userIdx[userID]
	if !ok {
		return []*outbound.Session{}, nil
	}

	sessions := make([]*outbound.Session, 0, len(userSessions))
	for sessionID := range userSessions {
		if session, exists := s.sessions[sessionID]; exists {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

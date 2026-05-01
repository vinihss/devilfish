package entity

import (
	"errors"
	"time"
)

// ErrInvalidEmbedding is returned when embedding data is invalid.
var ErrInvalidEmbedding = errors.New("invalid embedding")

// Embedding represents a vector embedding for a message, used for semantic search.
type Embedding struct {
	MessageID string    // ID of the source message
	SessionID string    // Session this embedding belongs to
	Vector    []float32 // Dense vector representation of the content
	Content   string    // Original text content that was embedded
	CreatedAt time.Time // When the embedding was created
}

// NewEmbedding creates a new Embedding entity.
func NewEmbedding(messageID, sessionID string, vector []float32, content string) (*Embedding, error) {
	if messageID == "" {
		return nil, errors.New("message id is required")
	}
	if sessionID == "" {
		return nil, errors.New("session id is required")
	}
	if len(vector) == 0 {
		return nil, errors.New("vector is required")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}

	return &Embedding{
		MessageID: messageID,
		SessionID: sessionID,
		Vector:    vector,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

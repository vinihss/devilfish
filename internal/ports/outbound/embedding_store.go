package outbound

import "context"

// StoredEmbedding represents an embedding entry in the store.
type StoredEmbedding struct {
	MessageID string
	SessionID string
	Vector    []float32
	Content   string
}

// EmbeddingStore persists and retrieves message embeddings for semantic search.
type EmbeddingStore interface {
	// Save persists a message embedding.
	Save(ctx context.Context, embedding *StoredEmbedding) error

	// SearchSimilar retrieves the top-K most semantically similar embeddings
	// within the given session, excluding the message with excludeID.
	SearchSimilar(ctx context.Context, sessionID string, vector []float32, topK int, excludeID string) ([]*StoredEmbedding, error)

	// DeleteBySession removes all embeddings belonging to a session.
	DeleteBySession(ctx context.Context, sessionID string) error
}

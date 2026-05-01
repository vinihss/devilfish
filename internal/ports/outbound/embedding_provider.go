package outbound

import "context"

// EmbeddingProvider generates vector embeddings for text content.
type EmbeddingProvider interface {
	// GenerateEmbedding converts text into a vector representation.
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// IsAvailable returns whether the embedding provider is available.
	IsAvailable() bool
}

package memory

import (
	"context"
	"math"
	"sort"
	"sync"

	"devilfish/internal/ports/outbound"
)

// EmbeddingStore is a thread-safe, in-memory implementation of
// outbound.EmbeddingStore. It uses cosine similarity for semantic search.
type EmbeddingStore struct {
	mu         sync.RWMutex
	embeddings []*outbound.StoredEmbedding          // flat list for iteration
	sessionIdx map[string][]*outbound.StoredEmbedding // sessionID → entries
}

// NewEmbeddingStore creates a new, empty in-memory EmbeddingStore.
func NewEmbeddingStore() *EmbeddingStore {
	return &EmbeddingStore{
		embeddings: make([]*outbound.StoredEmbedding, 0),
		sessionIdx: make(map[string][]*outbound.StoredEmbedding),
	}
}

// Save persists a message embedding.
func (s *EmbeddingStore) Save(_ context.Context, embedding *outbound.StoredEmbedding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.embeddings = append(s.embeddings, embedding)
	s.sessionIdx[embedding.SessionID] = append(s.sessionIdx[embedding.SessionID], embedding)
	return nil
}

// SearchSimilar returns the top-K embeddings within the given session that are
// most similar to the query vector, excluding the entry whose MessageID equals
// excludeID. Results are ranked by cosine similarity descending.
func (s *EmbeddingStore) SearchSimilar(_ context.Context, sessionID string, vector []float32, topK int, excludeID string) ([]*outbound.StoredEmbedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.sessionIdx[sessionID]
	if len(entries) == 0 || topK <= 0 {
		return []*outbound.StoredEmbedding{}, nil
	}

	type scored struct {
		embedding *outbound.StoredEmbedding
		score     float64
	}

	var candidates []scored
	for _, e := range entries {
		if e.MessageID == excludeID {
			continue
		}
		sim := cosineSimilarity(vector, e.Vector)
		candidates = append(candidates, scored{e, sim})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	k := topK
	if k > len(candidates) {
		k = len(candidates)
	}

	result := make([]*outbound.StoredEmbedding, k)
	for i := 0; i < k; i++ {
		result[i] = candidates[i].embedding
	}
	return result, nil
}

// DeleteBySession removes all embeddings belonging to a session.
func (s *EmbeddingStore) DeleteBySession(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from session index.
	delete(s.sessionIdx, sessionID)

	// Remove from the flat list.
	// Note: reusing the underlying array keeps allocations low. For production
	// use cases with very high session churn, consider a periodic reallocation.
	filtered := s.embeddings[:0]
	for _, e := range s.embeddings {
		if e.SessionID != sessionID {
			filtered = append(filtered, e)
		}
	}
	s.embeddings = filtered
	return nil
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 when either vector has zero magnitude to avoid division by zero.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, magA, magB float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		magA += ai * ai
		magB += bi * bi
	}

	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// Compile-time check that EmbeddingStore implements outbound.EmbeddingStore.
var _ outbound.EmbeddingStore = (*EmbeddingStore)(nil)

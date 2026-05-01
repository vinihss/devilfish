package storage_test

import (
	"context"
	"testing"

	"devilfish/internal/adapters/storage/memory"
	"devilfish/internal/ports/outbound"
)

func newEmbedding(msgID, sessionID string, vec []float32) *outbound.StoredEmbedding {
	return &outbound.StoredEmbedding{
		MessageID: msgID,
		SessionID: sessionID,
		Vector:    vec,
		Content:   "content for " + msgID,
	}
}

func TestEmbeddingStore_Save_And_SearchSimilar_ReturnsTopK(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("m1", "s1", []float32{1, 0, 0}))
	_ = store.Save(ctx, newEmbedding("m2", "s1", []float32{0, 1, 0}))
	_ = store.Save(ctx, newEmbedding("m3", "s1", []float32{0, 0, 1}))

	// Query vector is close to m1
	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0, 0}, 2, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].MessageID != "m1" {
		t.Errorf("expected m1 as top result, got %s", results[0].MessageID)
	}
}

func TestEmbeddingStore_SearchSimilar_ExcludesID(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("m1", "s1", []float32{1, 0, 0}))
	_ = store.Save(ctx, newEmbedding("m2", "s1", []float32{1, 0, 0}))

	// Exclude m1 — m2 should be returned.
	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0, 0}, 5, "m1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range results {
		if r.MessageID == "m1" {
			t.Error("expected m1 to be excluded")
		}
	}
	if len(results) != 1 || results[0].MessageID != "m2" {
		t.Errorf("expected [m2], got %v", results)
	}
}

func TestEmbeddingStore_SearchSimilar_WithZeroTopK_ReturnsEmpty(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()
	_ = store.Save(ctx, newEmbedding("m1", "s1", []float32{1, 0}))

	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0}, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestEmbeddingStore_SearchSimilar_DifferentSessions_Isolated(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("m1", "session-A", []float32{1, 0}))
	_ = store.Save(ctx, newEmbedding("m2", "session-B", []float32{1, 0}))

	results, err := store.SearchSimilar(ctx, "session-A", []float32{1, 0}, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for session-A, got %d", len(results))
	}
	if results[0].MessageID != "m1" {
		t.Errorf("expected m1 for session-A, got %s", results[0].MessageID)
	}
}

func TestEmbeddingStore_DeleteBySession_RemovesAllEntries(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("m1", "s1", []float32{1, 0}))
	_ = store.Save(ctx, newEmbedding("m2", "s1", []float32{0, 1}))
	_ = store.Save(ctx, newEmbedding("m3", "s2", []float32{1, 0}))

	if err := store.DeleteBySession(ctx, "s1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, _ := store.SearchSimilar(ctx, "s1", []float32{1, 0}, 10, "")
	if len(results) != 0 {
		t.Errorf("expected 0 results for deleted session, got %d", len(results))
	}

	// s2 should be unaffected.
	results2, _ := store.SearchSimilar(ctx, "s2", []float32{1, 0}, 10, "")
	if len(results2) != 1 {
		t.Errorf("expected 1 result for s2, got %d", len(results2))
	}
}

func TestEmbeddingStore_SearchSimilar_EmptyStore_ReturnsEmpty(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0}, 5, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestEmbeddingStore_SearchSimilar_TopKCappedByAvailable(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("m1", "s1", []float32{1, 0}))
	_ = store.Save(ctx, newEmbedding("m2", "s1", []float32{0, 1}))

	// Request more than available.
	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0}, 100, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (capped), got %d", len(results))
	}
}

func TestEmbeddingStore_CosineSimilarity_OrthogonalVectors_LowestScore(t *testing.T) {
	store := memory.NewEmbeddingStore()
	ctx := context.Background()

	_ = store.Save(ctx, newEmbedding("parallel", "s1", []float32{1, 0}))
	_ = store.Save(ctx, newEmbedding("orthogonal", "s1", []float32{0, 1}))

	// Query [1,0] should rank "parallel" above "orthogonal"
	results, err := store.SearchSimilar(ctx, "s1", []float32{1, 0}, 2, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].MessageID != "parallel" {
		t.Errorf("expected parallel vector to rank first, got %s", results[0].MessageID)
	}
}

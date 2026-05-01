package entity_test

import (
	"testing"

	"devilfish/internal/domain/entity"
)

func TestNewEmbedding_WithValidInput_ReturnsEmbedding(t *testing.T) {
	vec := []float32{0.1, 0.2, 0.3}

	emb, err := entity.NewEmbedding("msg-001", "session-001", vec, "hello world")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected embedding, got nil")
	}
	if emb.MessageID != "msg-001" {
		t.Errorf("expected MessageID msg-001, got %s", emb.MessageID)
	}
	if emb.SessionID != "session-001" {
		t.Errorf("expected SessionID session-001, got %s", emb.SessionID)
	}
	if len(emb.Vector) != 3 {
		t.Errorf("expected vector length 3, got %d", len(emb.Vector))
	}
	if emb.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %s", emb.Content)
	}
	if emb.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewEmbedding_WithEmptyMessageID_ReturnsError(t *testing.T) {
	_, err := entity.NewEmbedding("", "session-001", []float32{0.1}, "content")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewEmbedding_WithEmptySessionID_ReturnsError(t *testing.T) {
	_, err := entity.NewEmbedding("msg-001", "", []float32{0.1}, "content")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewEmbedding_WithEmptyVector_ReturnsError(t *testing.T) {
	_, err := entity.NewEmbedding("msg-001", "session-001", []float32{}, "content")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewEmbedding_WithEmptyContent_ReturnsError(t *testing.T) {
	_, err := entity.NewEmbedding("msg-001", "session-001", []float32{0.1}, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

package entity_test

import (
	"testing"

	"devilfish/internal/domain/entity"
)

func TestNewMessage_WithValidInput_ReturnsMessage(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.ID != "msg-001" {
		t.Errorf("expected ID msg-001, got %s", msg.ID)
	}
	if msg.Content != "Hello" {
		t.Errorf("expected Content Hello, got %s", msg.Content)
	}
	if msg.UserID != "user-001" {
		t.Errorf("expected UserID user-001, got %s", msg.UserID)
	}
	if msg.Channel != "telegram" {
		t.Errorf("expected Channel telegram, got %s", msg.Channel)
	}
	if msg.Type != entity.MessageTypeText {
		t.Errorf("expected Type MessageTypeText, got %v", msg.Type)
	}
	if msg.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}
}

func TestNewMessage_WithEmptyID_ReturnsError(t *testing.T) {
	msg, err := entity.NewMessage("", "Hello", "user-001", "telegram", entity.MessageTypeText)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
	if err.Error() != "message id is required" {
		t.Errorf("expected 'message id is required', got %s", err.Error())
	}
}

func TestNewMessage_WithEmptyContent_ReturnsError(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "", "user-001", "telegram", entity.MessageTypeText)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
	if err.Error() != "message content is required" {
		t.Errorf("expected 'message content is required', got %s", err.Error())
	}
}

func TestNewMessage_WithEmptyUserID_ReturnsError(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "", "telegram", entity.MessageTypeText)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
	if err.Error() != "user id is required" {
		t.Errorf("expected 'user id is required', got %s", err.Error())
	}
}

func TestNewMessage_WithEmptyChannel_ReturnsError(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "user-001", "", entity.MessageTypeText)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
	if err.Error() != "channel is required" {
		t.Errorf("expected 'channel is required', got %s", err.Error())
	}
}

func TestNewMessage_WithInvalidMessageType_ReturnsError(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", "invalid")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
	if err != entity.ErrInvalidMessageType {
		t.Errorf("expected ErrInvalidMessageType, got %v", err)
	}
}

func TestNewMessage_WithImageType_ReturnsMessage(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Image URL", "user-001", "discord", entity.MessageTypeImage)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Type != entity.MessageTypeImage {
		t.Errorf("expected Type MessageTypeImage, got %v", msg.Type)
	}
}

func TestNewMessage_WithAudioType_ReturnsMessage(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Audio", "user-001", "slack", entity.MessageTypeAudio)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Type != entity.MessageTypeAudio {
		t.Errorf("expected Type MessageTypeAudio, got %v", msg.Type)
	}
}

func TestNewMessage_WithVideoType_ReturnsMessage(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Video", "user-001", "telegram", entity.MessageTypeVideo)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Type != entity.MessageTypeVideo {
		t.Errorf("expected Type MessageTypeVideo, got %v", msg.Type)
	}
}

func TestNewMessage_WithDocumentType_ReturnsMessage(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Document", "user-001", "telegram", entity.MessageTypeDocument)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Type != entity.MessageTypeDocument {
		t.Errorf("expected Type MessageTypeDocument, got %v", msg.Type)
	}
}

func TestMessage_SetMetadata_SetsValue(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg.SetMetadata("key", "value")
	val, ok := msg.GetMetadata("key")

	if !ok {
		t.Error("expected key to exist")
	}
	if val != "value" {
		t.Errorf("expected value, got %v", val)
	}
}

func TestMessage_GetMetadata_WithNonExistentKey_ReturnsFalse(t *testing.T) {
	msg, err := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := msg.GetMetadata("non-existent")

	if ok {
		t.Error("expected key to not exist")
	}
}

func TestMessage_GetMetadata_WithNilMetadata_ReturnsFalse(t *testing.T) {
	msg := &entity.Message{
		ID:      "msg-001",
		Content: "Hello",
		UserID:  "user-001",
		Channel: "telegram",
		Type:    entity.MessageTypeText,
	}

	_, ok := msg.GetMetadata("key")

	if ok {
		t.Error("expected key to not exist")
	}
}
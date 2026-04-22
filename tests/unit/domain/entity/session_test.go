package entity_test

import (
	"testing"
	"time"

	"devilfish/internal/domain/entity"
)

func TestNewSession_WithValidInput_ReturnsSession(t *testing.T) {
	session, err := entity.NewSession("session-001", "user-001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected session, got nil")
	}
	if session.ID != "session-001" {
		t.Errorf("expected ID session-001, got %s", session.ID)
	}
	if session.UserID != "user-001" {
		t.Errorf("expected UserID user-001, got %s", session.UserID)
	}
	if session.Messages == nil {
		t.Error("expected Messages to be initialized")
	}
	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}
	if session.Context == nil {
		t.Error("expected Context to be initialized")
	}
	if !session.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestNewSession_WithEmptyID_ReturnsError(t *testing.T) {
	session, err := entity.NewSession("", "user-001")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if session != nil {
		t.Errorf("expected nil session, got %v", session)
	}
	if err.Error() != "session id is required" {
		t.Errorf("expected 'session id is required', got %s", err.Error())
	}
}

func TestNewSession_WithEmptyUserID_ReturnsError(t *testing.T) {
	session, err := entity.NewSession("session-001", "")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if session != nil {
		t.Errorf("expected nil session, got %v", session)
	}
	if err.Error() != "user id is required" {
		t.Errorf("expected 'user id is required', got %s", err.Error())
	}
}

func TestSession_AddMessage_AppendsMessage(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	msg, _ := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)

	err := session.AddMessage(msg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", session.MessageCount())
	}
}

func TestSession_AddMessage_WithNilMessage_ReturnsError(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")

	err := session.AddMessage(nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != entity.ErrInvalidSession {
		t.Errorf("expected ErrInvalidSession, got %v", err)
	}
}

func TestSession_ClearHistory_RemovesAllMessages(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	msg, _ := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)
	session.AddMessage(msg)
	session.AddMessage(msg)

	session.ClearHistory()

	if session.MessageCount() != 0 {
		t.Errorf("expected 0 messages, got %d", session.MessageCount())
	}
}

func TestSession_MessageCount_ReturnsCorrectCount(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	msg, _ := entity.NewMessage("msg-001", "Hello", "user-001", "telegram", entity.MessageTypeText)

	if session.MessageCount() != 0 {
		t.Errorf("expected 0 messages initially, got %d", session.MessageCount())
	}

	session.AddMessage(msg)
	if session.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", session.MessageCount())
	}

	session.AddMessage(msg)
	if session.MessageCount() != 2 {
		t.Errorf("expected 2 messages, got %d", session.MessageCount())
	}
}

func TestSession_SetContext_UpdatesContext(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	ctx := &entity.SessionContext{
		ProviderName: "openai",
		Model:         "gpt-4",
	}

	err := session.SetContext(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Context.ProviderName != "openai" {
		t.Errorf("expected provider openai, got %s", session.Context.ProviderName)
	}
	if session.Context.Model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", session.Context.Model)
	}
}

func TestSession_SetContext_WithNilContext_ReturnsError(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")

	err := session.SetContext(nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != entity.ErrInvalidSession {
		t.Errorf("expected ErrInvalidSession, got %v", err)
	}
}

func TestSession_SetProvider_SetsProvider(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")

	session.SetProvider("groq")

	if session.Provider == nil {
		t.Error("expected Provider to be set")
	}
	if *session.Provider != "groq" {
		t.Errorf("expected groq, got %s", *session.Provider)
	}
}

func TestSession_Deactivate_MarksInactive(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	originalTime := session.UpdatedAt
	time.Sleep(time.Millisecond)

	session.Deactivate()

	if session.IsActive {
		t.Error("expected IsActive to be false")
	}
	if !session.UpdatedAt.After(originalTime) && !session.UpdatedAt.Equal(originalTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSession_Activate_MarksActive(t *testing.T) {
	session, _ := entity.NewSession("session-001", "user-001")
	session.Deactivate()
	originalTime := session.UpdatedAt
	time.Sleep(time.Millisecond)

	session.Activate()

	if !session.IsActive {
		t.Error("expected IsActive to be true")
	}
	if !session.UpdatedAt.After(originalTime) && !session.UpdatedAt.Equal(originalTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}
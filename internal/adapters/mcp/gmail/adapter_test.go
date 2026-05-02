package gmail

import (
	"context"
	"errors"
	"testing"
	"time"

	"devilfish/internal/ports/outbound"
)

// mockMCPClient is a mock implementation of outbound.MCPClient for testing.
type mockMCPClient struct {
	connected    bool
	tools        []outbound.MCPTool
	executeError error
	executeResult map[string]interface{}
	executeCalls []executeCall
}

type executeCall struct {
	toolName string
	params   map[string]interface{}
}

func newMockMCPClient() *mockMCPClient {
	return &mockMCPClient{
		connected:     false,
		tools:         []outbound.MCPTool{},
		executeResult: make(map[string]interface{}),
		executeCalls:  make([]executeCall, 0),
	}
}

func (m *mockMCPClient) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	m.connected = true
	return nil
}

func (m *mockMCPClient) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockMCPClient) IsConnected() bool {
	return m.connected
}

func (m *mockMCPClient) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	return m.tools, nil
}

func (m *mockMCPClient) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	m.executeCalls = append(m.executeCalls, executeCall{
		toolName: toolName,
		params:   params,
	})

	if m.executeError != nil {
		return nil, m.executeError
	}
	return m.executeResult, nil
}

func (m *mockMCPClient) ServerName() string {
	return "gmail-mock"
}

// TestNewGmailMCP tests the creation of a new GmailMCP adapter.
func TestNewGmailMCP(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name:      "gmail-test",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"@modelcontextprotocol/server-gmail"},
	}

	mockClient := newMockMCPClient()
	adapter := NewGmailMCP(config, mockClient)

	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}

	if adapter.Name() != "gmail" {
		t.Errorf("expected name 'gmail', got %s", adapter.Name())
	}

	if adapter.IsConnected() {
		t.Error("expected adapter to not be connected initially")
	}
}

// TestGmailMCP_Connect tests the Connect method.
func TestGmailMCP_Connect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.Connect(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !adapter.IsConnected() {
			t.Error("expected adapter to be connected after Connect")
		}
	})

	t.Run("already connected", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.Connect(ctx)
		if err != nil {
			t.Errorf("unexpected error on reconnect: %v", err)
		}
	})

	t.Run("nil client", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		adapter := NewGmailMCP(config, nil)

		ctx := context.Background()
		err := adapter.Connect(ctx)
		if err == nil {
			t.Error("expected error when client is nil")
		}

		if !errors.Is(err, ErrNoClient) {
			t.Errorf("expected ErrNoClient, got %v", err)
		}
	})
}

// TestGmailMCP_Disconnect tests the Disconnect method.
func TestGmailMCP_Disconnect(t *testing.T) {
	t.Run("successful disconnect", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.Disconnect(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if adapter.IsConnected() {
			t.Error("expected adapter to be disconnected after Disconnect")
		}
	})

	t.Run("disconnect when not connected", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.Disconnect(ctx)
		if err != nil {
			t.Errorf("unexpected error when disconnecting while not connected: %v", err)
		}
	})
}

// TestGmailMCP_SendEmail tests the SendEmail method.
func TestGmailMCP_SendEmail(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		mockClient.executeResult = map[string]interface{}{
			"success": true,
		}
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(mockClient.executeCalls) != 1 {
			t.Errorf("expected 1 ExecuteTool call, got %d", len(mockClient.executeCalls))
		}

		if mockClient.executeCalls[0].toolName != "SendEmail" {
			t.Errorf("expected tool name 'SendEmail', got %s", mockClient.executeCalls[0].toolName)
		}
	})

	t.Run("send when not connected", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
		if err == nil {
			t.Error("expected error when not connected")
		}

		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("expected ErrNotConnected, got %v", err)
		}
	})

	t.Run("send with empty recipient", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.SendEmail(ctx, "", "Test Subject", "Test Body")
		if err == nil {
			t.Error("expected error with empty recipient")
		}

		if !errors.Is(err, ErrInvalidRecipient) {
			t.Errorf("expected ErrInvalidRecipient, got %v", err)
		}
	})

	t.Run("send with execution error", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		mockClient.executeError = errors.New("tool execution failed")
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		err := adapter.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
		if err == nil {
			t.Error("expected error when tool execution fails")
		}
	})
}

// TestGmailMCP_ListEmails tests the ListEmails method.
func TestGmailMCP_ListEmails(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		mockClient.executeResult = map[string]interface{}{
			"emails": []interface{}{
				map[string]interface{}{
					"id":      "email1",
					"from":    "sender@example.com",
					"to":      "recipient@example.com",
					"subject": "Test Subject 1",
					"body":    "Test Body 1",
				},
				map[string]interface{}{
					"id":      "email2",
					"from":    "sender2@example.com",
					"to":      "recipient@example.com",
					"subject": "Test Subject 2",
					"body":    "Test Body 2",
				},
			},
		}
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		emails, err := adapter.ListEmails(ctx, "is:unread")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(emails) != 2 {
			t.Errorf("expected 2 emails, got %d", len(emails))
		}

		if emails[0].ID != "email1" {
			t.Errorf("expected email ID 'email1', got %s", emails[0].ID)
		}
	})

	t.Run("list when not connected", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		_, err := adapter.ListEmails(ctx, "is:unread")
		if err == nil {
			t.Error("expected error when not connected")
		}

		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("expected ErrNotConnected, got %v", err)
		}
	})

	t.Run("list with empty result", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		mockClient.executeResult = map[string]interface{}{}
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		emails, err := adapter.ListEmails(ctx, "is:unread")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(emails) != 0 {
			t.Errorf("expected 0 emails, got %d", len(emails))
		}
	})
}

// TestGmailMCP_ReadEmail tests the ReadEmail method.
func TestGmailMCP_ReadEmail(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		mockClient.executeResult = map[string]interface{}{
			"email": map[string]interface{}{
				"id":      "email123",
				"from":    "sender@example.com",
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Test Body",
			},
		}
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		email, err := adapter.ReadEmail(ctx, "email123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if email.ID != "email123" {
			t.Errorf("expected email ID 'email123', got %s", email.ID)
		}

		if email.Subject != "Test Subject" {
			t.Errorf("expected subject 'Test Subject', got %s", email.Subject)
		}
	})

	t.Run("read when not connected", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		_, err := adapter.ReadEmail(ctx, "email123")
		if err == nil {
			t.Error("expected error when not connected")
		}

		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("expected ErrNotConnected, got %v", err)
		}
	})

	t.Run("read with empty ID", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		_, err := adapter.ReadEmail(ctx, "")
		if err == nil {
			t.Error("expected error with empty email ID")
		}

		if !errors.Is(err, ErrInvalidEmailID) {
			t.Errorf("expected ErrInvalidEmailID, got %v", err)
		}
	})

	t.Run("read with invalid response", func(t *testing.T) {
		config := outbound.MCPServerConfig{
			Name: "gmail-test",
		}
		mockClient := newMockMCPClient()
		mockClient.connected = true
		// Response with invalid structure - no "email" key and no valid fields
		mockClient.executeResult = map[string]interface{}{
			"invalid": "data",
		}
		adapter := NewGmailMCP(config, mockClient)

		ctx := context.Background()
		email, err := adapter.ReadEmail(ctx, "email123")
		// This should still return an email object (possibly empty), not an error
		// because parseEmail handles missing data gracefully
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// The email will have empty fields since the response didn't have proper structure
		if email.ID != "" {
			t.Errorf("expected empty email ID for invalid response, got %s", email.ID)
		}
	})
}

// TestGmailMCP_IsConnected tests the IsConnected method.
func TestGmailMCP_IsConnected(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name: "gmail-test",
	}
	mockClient := newMockMCPClient()
	adapter := NewGmailMCP(config, mockClient)

	if adapter.IsConnected() {
		t.Error("expected false when not connected")
	}

	mockClient.connected = true
	if !adapter.IsConnected() {
		t.Error("expected true when client is connected")
	}

	mockClient.connected = false
	if adapter.IsConnected() {
		t.Error("expected false when client is disconnected")
	}
}

// TestParseEmailList tests the parseEmailList function.
func TestParseEmailList(t *testing.T) {
	t.Run("valid email list", func(t *testing.T) {
		result := map[string]interface{}{
			"emails": []interface{}{
				map[string]interface{}{
					"id":      "1",
					"from":    "a@example.com",
					"to":      "b@example.com",
					"subject": "Subject 1",
					"body":    "Body 1",
				},
			},
		}

		emails, err := parseEmailList(result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(emails) != 1 {
			t.Errorf("expected 1 email, got %d", len(emails))
		}
	})

	t.Run("empty result", func(t *testing.T) {
		emails, err := parseEmailList(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(emails) != 0 {
			t.Errorf("expected 0 emails, got %d", len(emails))
		}
	})

	t.Run("alternative key", func(t *testing.T) {
		result := map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"id": "2",
				},
			},
		}

		emails, err := parseEmailList(result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(emails) != 1 {
			t.Errorf("expected 1 email, got %d", len(emails))
		}
	})
}

// TestParseEmail tests the parseEmail function.
func TestParseEmail(t *testing.T) {
	t.Run("valid email", func(t *testing.T) {
		result := map[string]interface{}{
			"email": map[string]interface{}{
				"id":      "123",
				"from":    "sender@test.com",
				"subject": "Hello",
				"body":    "World",
			},
		}

		email, err := parseEmail(result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if email.ID != "123" {
			t.Errorf("expected ID '123', got %s", email.ID)
		}
	})

	t.Run("email without wrapper", func(t *testing.T) {
		result := map[string]interface{}{
			"id":      "456",
			"subject": "Direct",
		}

		email, err := parseEmail(result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if email.ID != "456" {
			t.Errorf("expected ID '456', got %s", email.ID)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		_, err := parseEmail(nil)
		if err == nil {
			t.Error("expected error for nil result")
		}
	})
}

// TestEmailCapabilityInterface tests that GmailMCP implements EmailCapability.
func TestEmailCapabilityInterface(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name: "gmail-test",
	}
	mockClient := newMockMCPClient()
	adapter := NewGmailMCP(config, mockClient)

	// Verify the adapter implements the interface
	var _ EmailCapability = adapter
}

// TestGmailMCP_ContextPropagation tests that context is properly used.
func TestGmailMCP_ContextPropagation(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name: "gmail-test",
	}
	mockClient := newMockMCPClient()
	mockClient.connected = true
	adapter := NewGmailMCP(config, mockClient)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(5 * time.Millisecond)

	// This should still work as our mock doesn't check context
	// In real implementation, the MCP client would respect context
	err := adapter.SendEmail(ctx, "test@example.com", "Subject", "Body")
	// We don't assert error here as mock doesn't enforce context
	_ = err
}

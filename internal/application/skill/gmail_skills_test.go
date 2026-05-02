package skill

import (
	"context"
	"testing"

	"devilfish/internal/adapters/mcp"
	"github.com/stretchr/testify/assert"
)

// TestSendEmailSkill_Name tests the Name method.
func TestSendEmailSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewSendEmailSkill(registry)

	assert.Equal(t, "send_email", skill.Name())
}

// TestSendEmailSkill_Description tests the Description method.
func TestSendEmailSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewSendEmailSkill(registry)

	assert.Contains(t, skill.Description(), "Send an email")
}

// TestSendEmailSkill_Schema tests the Schema method.
func TestSendEmailSkill_Schema(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewSendEmailSkill(registry)

	schema := skill.Schema()
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, properties, "to")
	assert.Contains(t, properties, "subject")
	assert.Contains(t, properties, "body")

	required, ok := schema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "to")
	assert.Contains(t, required, "subject")
	assert.Contains(t, required, "body")
}

// TestSendEmailSkill_Execute_ServerNotFound tests when Gmail server is not found.
func TestSendEmailSkill_Execute_ServerNotFound(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewSendEmailSkill(registry)

	args := map[string]interface{}{
		"to":      "test@example.com",
		"subject": "Test Subject",
		"body":    "Test Body",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)
}

// TestSendEmailSkill_Execute_MissingTo tests when 'to' parameter is missing.
func TestSendEmailSkill_Execute_MissingTo(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewSendEmailSkill(registry)

	args := map[string]interface{}{
		"subject": "Test Subject",
		"body":    "Test Body",
		// Missing 'to'
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to")
	assert.Empty(t, result)
}

// TestSendEmailSkill_Execute_EmptyTo tests when 'to' parameter is empty.
func TestSendEmailSkill_Execute_EmptyTo(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewSendEmailSkill(registry)

	args := map[string]interface{}{
		"to":      "",
		"subject": "Test Subject",
		"body":    "Test Body",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to")
	assert.Empty(t, result)
}

// TestListEmailsSkill_Name tests the Name method.
func TestListEmailsSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewListEmailsSkill(registry)

	assert.Equal(t, "list_emails", skill.Name())
}

// TestListEmailsSkill_Description tests the Description method.
func TestListEmailsSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewListEmailsSkill(registry)

	assert.Contains(t, skill.Description(), "List emails")
}

// TestListEmailsSkill_Execute_ServerNotFound tests when Gmail server is not found.
func TestListEmailsSkill_Execute_ServerNotFound(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewListEmailsSkill(registry)

	args := map[string]interface{}{
		"query": "is:unread",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)
}

// TestListEmailsSkill_Execute_NoQuery uses default query.
func TestListEmailsSkill_Execute_NoQuery(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewListEmailsSkill(registry)

	args := map[string]interface{}{
		// No query provided
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)
}

// TestReadEmailSkill_Name tests the Name method.
func TestReadEmailSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewReadEmailSkill(registry)

	assert.Equal(t, "read_email", skill.Name())
}

// TestReadEmailSkill_Description tests the Description method.
func TestReadEmailSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewReadEmailSkill(registry)

	assert.Contains(t, skill.Description(), "Read a specific email")
}

// TestReadEmailSkill_Execute_MissingID tests when 'id' parameter is missing.
func TestReadEmailSkill_Execute_MissingID(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewReadEmailSkill(registry)

	args := map[string]interface{}{
		// Missing 'id'
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")
	assert.Empty(t, result)
}

// TestReadEmailSkill_Execute_EmptyID tests when 'id' parameter is empty.
func TestReadEmailSkill_Execute_EmptyID(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewReadEmailSkill(registry)

	args := map[string]interface{}{
		"id": "",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")
	assert.Empty(t, result)
}

// TestGmailSkills_InterfaceCompliance tests that all skills implement the Skill interface.
func TestGmailSkills_InterfaceCompliance(t *testing.T) {
	registry := mcp.NewRegistry(nil)

	// Test SendEmailSkill
	sendSkill := NewSendEmailSkill(registry)
	assert.Implements(t, (*Skill)(nil), sendSkill)

	// Test ListEmailsSkill
	listSkill := NewListEmailsSkill(registry)
	assert.Implements(t, (*Skill)(nil), listSkill)

	// Test ReadEmailSkill
	readSkill := NewReadEmailSkill(registry)
	assert.Implements(t, (*Skill)(nil), readSkill)
}

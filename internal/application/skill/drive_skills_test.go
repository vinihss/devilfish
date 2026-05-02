package skill

import (
	"context"
	"testing"

	"devilfish/internal/adapters/mcp"
	"github.com/stretchr/testify/assert"
)

// TestListFilesSkill_Name tests the Name method.
func TestListFilesSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewListFilesSkill(registry)

	assert.Equal(t, "list_files", skill.Name())
}

// TestListFilesSkill_Description tests the Description method.
func TestListFilesSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewListFilesSkill(registry)

	assert.Contains(t, skill.Description(), "List files")
}

// TestListFilesSkill_Schema tests the Schema method.
func TestListFilesSkill_Schema(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewListFilesSkill(registry)

	schema := skill.Schema()
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, properties, "query")

	required, ok := schema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "query")
}

// TestListFilesSkill_Execute_ServerNotFound tests when Drive server is not found.
func TestListFilesSkill_Execute_ServerNotFound(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewListFilesSkill(registry)

	args := map[string]interface{}{
		"query": "test",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)
}

// TestReadFileSkill_Name tests the Name method.
func TestReadFileSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewReadFileSkill(registry)

	assert.Equal(t, "read_file", skill.Name())
}

// TestReadFileSkill_Description tests the Description method.
func TestReadFileSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewReadFileSkill(registry)

	assert.Contains(t, skill.Description(), "Read the content")
}

// TestReadFileSkill_Execute_MissingFileID tests when 'file_id' parameter is missing.
func TestReadFileSkill_Execute_MissingFileID(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewReadFileSkill(registry)

	args := map[string]interface{}{
		// Missing 'file_id'
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_id")
	assert.Empty(t, result)
}

// TestReadFileSkill_Execute_EmptyFileID tests when 'file_id' is empty.
func TestReadFileSkill_Execute_EmptyFileID(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewReadFileSkill(registry)

	args := map[string]interface{}{
		"file_id": "",
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_id")
	assert.Empty(t, result)
}

// TestGetFileInfoSkill_Name tests the Name method.
func TestGetFileInfoSkill_Name(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewGetFileInfoSkill(registry)

	assert.Equal(t, "get_file_info", skill.Name())
}

// TestGetFileInfoSkill_Description tests the Description method.
func TestGetFileInfoSkill_Description(t *testing.T) {
	registry := mcp.NewRegistry(nil)
	skill := NewGetFileInfoSkill(registry)

	assert.Contains(t, skill.Description(), "metadata")
}

// TestGetFileInfoSkill_Execute_MissingFileID tests when 'file_id' parameter is missing.
func TestGetFileInfoSkill_Execute_MissingFileID(t *testing.T) {
	ctx := context.Background()
	registry := mcp.NewRegistry(nil)

	skill := NewGetFileInfoSkill(registry)

	args := map[string]interface{}{
		// Missing 'file_id'
	}

	result, err := skill.Execute(ctx, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_id")
	assert.Empty(t, result)
}

// TestDriveSkills_InterfaceCompliance tests that all skills implement the Skill interface.
func TestDriveSkills_InterfaceCompliance(t *testing.T) {
	registry := mcp.NewRegistry(nil)

	// Test ListFilesSkill
	listSkill := NewListFilesSkill(registry)
	assert.Implements(t, (*Skill)(nil), listSkill)

	// Test ReadFileSkill
	readSkill := NewReadFileSkill(registry)
	assert.Implements(t, (*Skill)(nil), readSkill)

	// Test GetFileInfoSkill
	infoSkill := NewGetFileInfoSkill(registry)
	assert.Implements(t, (*Skill)(nil), infoSkill)
}

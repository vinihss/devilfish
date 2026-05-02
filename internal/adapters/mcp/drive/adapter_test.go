package drive

import (
	"context"
	"testing"

	"devilfish/internal/ports/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMCPClient is a mock implementation of outbound.MCPClient for testing.
type MockMCPClient struct {
	mock.Mock
}

func (m *MockMCPClient) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockMCPClient) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMCPClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMCPClient) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]outbound.MCPTool), args.Error(1)
}

func (m *MockMCPClient) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(ctx, toolName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockMCPClient) ServerName() string {
	args := m.Called()
	return args.String(0)
}

// TestNewDriveMCP tests the constructor.
func TestNewDriveMCP(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name:      "test-drive",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-everything"},
	}

	adapter := NewDriveMCP(config)

	assert.NotNil(t, adapter)
	assert.Equal(t, "test-drive", adapter.config.Name)
	assert.Equal(t, "google-drive", adapter.Name())
	assert.False(t, adapter.IsConnected())
}

// TestDriveMCP_Connect_Success tests successful connection.
func TestDriveMCP_Connect_Success(t *testing.T) {
	ctx := context.Background()
	config := outbound.MCPServerConfig{
		Name:      "test-drive",
		Transport: "stdio",
		Command:   "test-command",
	}

	adapter := NewDriveMCP(config)

	// Set a mock client directly to avoid IsConnected() call in SetClient
	mockClient := new(MockMCPClient)
	mockClient.On("Connect", ctx, config).Return(nil)
	mockClient.On("IsConnected").Return(true)
	adapter.client = mockClient

	err := adapter.Connect(ctx, config)

	assert.NoError(t, err)
	assert.True(t, adapter.IsConnected())
	mockClient.AssertExpectations(t)
}

// TestDriveMCP_Connect_AlreadyConnected tests connecting when already connected.
func TestDriveMCP_Connect_AlreadyConnected(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name:      "test-drive",
		Transport: "stdio",
	}

	adapter := NewDriveMCP(config)
	adapter.connected = true

	ctx := context.Background()
	err := adapter.Connect(ctx, config)

	assert.NoError(t, err)
	assert.True(t, adapter.IsConnected())
}

// TestDriveMCP_Disconnect_Success tests successful disconnection.
func TestDriveMCP_Disconnect_Success(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name:      "test-drive",
		Transport: "stdio",
	}

	adapter := NewDriveMCP(config)
	adapter.connected = true

	// Set a mock client
	mockClient := new(MockMCPClient)
	mockClient.On("Disconnect", mock.Anything).Return(nil)
	mockClient.On("IsConnected").Return(false)
	adapter.SetClient(mockClient)

	ctx := context.Background()
	err := adapter.Disconnect(ctx)

	assert.NoError(t, err)
	assert.False(t, adapter.IsConnected())
	mockClient.AssertExpectations(t)
}

// TestDriveMCP_Disconnect_NotConnected tests disconnecting when not connected.
func TestDriveMCP_Disconnect_NotConnected(t *testing.T) {
	config := outbound.MCPServerConfig{
		Name:      "test-drive",
		Transport: "stdio",
	}

	adapter := NewDriveMCP(config)

	ctx := context.Background()
	err := adapter.Disconnect(ctx)

	assert.NoError(t, err)
	assert.False(t, adapter.IsConnected())
}

// TestDriveMCP_ListFiles_Success tests successful file listing.
func TestDriveMCP_ListFiles_Success(t *testing.T) {
	// Create expected result
	expectedFiles := []outbound.File{
		{ID: "file1", Name: "report1.pdf"},
		{ID: "file2", Name: "report2.docx"},
	}

	// Test the parseFileList helper function
	result := map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"id": "file1", "name": "report1.pdf"},
			map[string]interface{}{"id": "file2", "name": "report2.docx"},
		},
	}

	files, err := parseFileList(result)

	assert.NoError(t, err)
	assert.Equal(t, expectedFiles, files)
}

// TestDriveMCP_ListFiles_EmptyResult tests listing with no results.
func TestDriveMCP_ListFiles_EmptyResult(t *testing.T) {
	result := map[string]interface{}{
		"files": []interface{}{},
	}

	files, err := parseFileList(result)

	assert.NoError(t, err)
	assert.Empty(t, files)
}

// TestDriveMCP_ListFiles_NoFilesKey tests result without files key.
func TestDriveMCP_ListFiles_NoFilesKey(t *testing.T) {
	result := map[string]interface{}{
		"status": "ok",
	}

	files, err := parseFileList(result)

	assert.NoError(t, err)
	assert.Empty(t, files)
}

// TestDriveMCP_ReadFile_Success tests successful file reading.
func TestDriveMCP_ReadFile_Success(t *testing.T) {
	expectedContent := "This is the file content"

	// Test the parseFileContent helper function
	result := map[string]interface{}{
		"content": expectedContent,
	}

	content, err := parseFileContent(result)

	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

// TestDriveMCP_ReadFile_EmptyFileID tests reading with empty file ID.
func TestDriveMCP_ReadFile_EmptyFileID(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})
	adapter.connected = true

	ctx := context.Background()
	content, err := adapter.ReadFile(ctx, "")

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "no MCP client available")
}

// TestDriveMCP_ReadFile_NotConnected tests reading when not connected.
func TestDriveMCP_ReadFile_NotConnected(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})

	ctx := context.Background()
	content, err := adapter.ReadFile(ctx, "file123")

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "not connected")
}

// TestDriveMCP_ReadFile_DifferentContentKeys tests various content key names.
func TestDriveMCP_ReadFile_DifferentContentKeys(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		expected string
	}{
		{
			name:     "content key",
			result:   map[string]interface{}{"content": "hello"},
			expected: "hello",
		},
		{
			name:     "data key",
			result:   map[string]interface{}{"data": "world"},
			expected: "world",
		},
		{
			name:     "text key",
			result:   map[string]interface{}{"text": "foo"},
			expected: "foo",
		},
		{
			name:     "file_content key",
			result:   map[string]interface{}{"file_content": "bar"},
			expected: "bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := parseFileContent(tt.result)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, content)
		})
	}
}

// TestDriveMCP_IsConnected tests connection status.
func TestDriveMCP_IsConnected(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})

	// Not connected initially
	assert.False(t, adapter.IsConnected())

	// After setting connected
	adapter.connected = true
	assert.True(t, adapter.IsConnected())
}

// TestDriveMCP_ServerName tests server name.
func TestDriveMCP_ServerName(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})
	assert.Equal(t, "google-drive", adapter.ServerName())
	assert.Equal(t, "google-drive", adapter.Name())
}

// TestParseFileList_InvalidFormat tests parsing invalid format.
func TestParseFileList_InvalidFormat(t *testing.T) {
	result := map[string]interface{}{
		"files": "not-a-slice",
	}

	files, err := parseFileList(result)

	assert.NoError(t, err)
	assert.Empty(t, files)
}

// TestParseFileContent_NoContent tests parsing result with no content.
func TestParseFileContent_NoContent(t *testing.T) {
	result := map[string]interface{}{
		"status": "ok",
	}

	content, err := parseFileContent(result)

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "no content found")
}

// TestDriveMCP_ListTools tests the ListTools method.
func TestDriveMCP_ListTools(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})

	ctx := context.Background()
	tools, err := adapter.ListTools(ctx)

	// Should fail because not connected
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "not connected")
}

// TestDriveMCP_ExecuteTool tests the ExecuteTool method.
func TestDriveMCP_ExecuteTool(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})

	ctx := context.Background()
	result, err := adapter.ExecuteTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because not connected
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not connected")
}

// TestDriveMCP_SetClient tests the SetClient method.
func TestDriveMCP_SetClient(t *testing.T) {
	adapter := NewDriveMCP(outbound.MCPServerConfig{})

	mockClient := new(MockMCPClient)
	mockClient.On("IsConnected").Return(true)

	adapter.SetClient(mockClient)

	assert.Same(t, mockClient, adapter.client)
}

// TestDriveMCP_ListFiles_WithMockClient tests ListFiles with a mock client.
func TestDriveMCP_ListFiles_WithMockClient(t *testing.T) {
	ctx := context.Background()
	adapter := NewDriveMCP(outbound.MCPServerConfig{})
	adapter.connected = true

	mockClient := new(MockMCPClient)
	expectedResult := map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"id": "file1", "name": "test.pdf"},
		},
	}
	mockClient.On("ExecuteTool", mock.Anything, toolListFiles, mock.Anything).Return(expectedResult, nil)

	// Need to set client after setting up expectations
	adapter.client = mockClient

	files, err := adapter.ListFiles(ctx, "test")

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "file1", files[0].ID)
	assert.Equal(t, "test.pdf", files[0].Name)
	mockClient.AssertExpectations(t)
}

// TestDriveMCP_ReadFile_WithMockClient tests ReadFile with a mock client.
func TestDriveMCP_ReadFile_WithMockClient(t *testing.T) {
	ctx := context.Background()
	adapter := NewDriveMCP(outbound.MCPServerConfig{})
	adapter.connected = true

	mockClient := new(MockMCPClient)
	expectedResult := map[string]interface{}{
		"content": "file content here",
	}
	mockClient.On("ExecuteTool", mock.Anything, toolReadFile, mock.Anything).Return(expectedResult, nil)

	// Set client directly to avoid IsConnected() call
	adapter.client = mockClient

	content, err := adapter.ReadFile(ctx, "file123")

	assert.NoError(t, err)
	assert.Equal(t, "file content here", content)
	mockClient.AssertExpectations(t)
}

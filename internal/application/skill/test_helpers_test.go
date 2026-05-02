package skill

import (
	"context"

	"devilfish/internal/ports/outbound"
	"github.com/stretchr/testify/mock"
)

// MockMCPClient is a mock implementation of outbound.MCPClient for testing.
// This file contains shared mocks used across multiple test files.
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

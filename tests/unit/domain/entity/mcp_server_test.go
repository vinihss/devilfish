package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"devilfish/internal/domain/entity"
)

func TestMCPServer_NewMCPServer(t *testing.T) {
	tests := []struct {
		name     string
		id      string
		cfg     entity.MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio server",
			id:   "server-1",
			cfg: entity.MCPServerConfig{
				Name:      "test-server",
				Transport: "stdio",
				Command:  "npx",
				Args:     []string{"-y", "some-package"},
				Timeout:  30,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid http server",
			id:   "server-2",
			cfg: entity.MCPServerConfig{
				Name:      "http-server",
				Transport: "http",
				URL:      "http://localhost:3000",
				Timeout:  60,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name:    "empty id",
			id:     "",
			cfg:     entity.MCPServerConfig{Name: "test"},
			wantErr: true,
			errMsg: "server id is required",
		},
		{
			name:    "empty name",
			id:     "server-1",
			cfg:    entity.MCPServerConfig{Transport: "stdio"},
			wantErr: true,
			errMsg: "server name is required",
		},
		{
			name: "default values",
			id:   "server-3",
			cfg: entity.MCPServerConfig{
				Name:      "test",
				Transport: "stdio",
				Enabled:  true, // Must be explicitly enabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := entity.NewMCPServer(tt.id, tt.cfg.Name, tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, server)
			assert.Equal(t, tt.id, server.ID)
			assert.Equal(t, tt.cfg.Name, server.Name)
			assert.True(t, server.Enabled)

			// Check defaults
			if tt.cfg.Timeout == 0 {
				assert.Equal(t, 30, server.Timeout)
			}
			if tt.cfg.MaxRetries == 0 {
				assert.Equal(t, 3, server.MaxRetries)
			}
		})
	}
}

func TestMCPServer_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		server  *entity.MCPServer
		wantErr bool
	}{
		{
			name: "valid stdio server",
			server: &entity.MCPServer{
				Name:     "test",
				Transport: "stdio",
				Command: "npx",
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "valid http server",
			server: &entity.MCPServer{
				Name:     "test",
				Transport: "http",
				URL:     "http://localhost:3000",
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "disabled server",
			server: &entity.MCPServer{
				Name:     "test",
				Transport: "stdio",
				Command: "npx",
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			server: &entity.MCPServer{
				Name:     "",
				Transport: "stdio",
				Command: "npx",
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "http without url",
			server: &entity.MCPServer{
				Name:     "test",
				Transport: "http",
				URL:     "",
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "stdio without command",
			server: &entity.MCPServer{
				Name:     "test",
				Transport: "stdio",
				Command: "",
				Enabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.server.IsValid()
			if tt.wantErr {
				assert.False(t, isValid)
			} else {
				assert.True(t, isValid)
			}
		})
	}
}

func TestMCPServer_EnableDisable(t *testing.T) {
	server := &entity.MCPServer{
		Name:     "test",
		Transport: "stdio",
		Command: "npx",
		Enabled: false,
	}

	assert.False(t, server.Enabled)

	server.Enable()
	assert.True(t, server.Enabled)
	assert.NotZero(t, server.UpdatedAt)

	server.Disable()
	assert.False(t, server.Enabled)
}

func TestMCPServer_SetAuthToken(t *testing.T) {
	server := &entity.MCPServer{
		Name:     "test",
		Transport: "http",
		URL:     "http://localhost:3000",
	}

	token := "secret-token-123"
	server.SetAuthToken(token)

	assert.Equal(t, token, server.AuthToken)
}

func TestMCPServer_GetConfig(t *testing.T) {
	cfg := entity.MCPServerConfig{
		Name:      "test",
		URL:      "http://localhost:3000",
		Transport: "http",
		Timeout:  30,
		MaxRetries: 3,
	}

	server, err := entity.NewMCPServer("s1", cfg.Name, cfg)
	assert.NoError(t, err)

 returnedCfg := server.GetConfig()
	assert.Equal(t, cfg.Name, returnedCfg.Name)
	assert.Equal(t, cfg.Transport, returnedCfg.Transport)
}
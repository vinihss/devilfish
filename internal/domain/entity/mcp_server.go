package entity

import (
	"errors"
	"time"
)

// MCPServerConfig entity configuration.
type MCPServerConfig struct {
	Name       string            `yaml:"name"`
	URL        string            `yaml:"url"`
	Transport string            `yaml:"transport"`
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args"`
	AuthToken string            `yaml:"auth_token"`
	Enabled  bool              `yaml:"enabled"`
	Timeout  int               `yaml:"timeout"`
	MaxRetries int             `yaml:"max_retries"`
}

// MCPServer represents an MCP server entity.
type MCPServer struct {
	ID          string             `json:"id"`
	Name       string             `json:"name"`
	URL        string             `json:"url,omitempty"`
	Transport string             `json:"transport"`
	Command   string             `json:"command,omitempty"`
	Args      []string           `json:"args,omitempty"`
	AuthToken string             `json:"-"`
	Enabled  bool               `json:"enabled"`
	Timeout  int                `json:"timeout"`
	MaxRetries int               `json:"max_retries"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// NewMCPServer creates a new MCPServer entity.
func NewMCPServer(id, name string, config MCPServerConfig) (*MCPServer, error) {
	if id == "" {
		return nil, errors.New("server id is required")
	}
	if name == "" {
		return nil, errors.New("server name is required")
	}

	transport := config.Transport
	if transport == "" {
		transport = "stdio"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &MCPServer{
		ID:         id,
		Name:       name,
		URL:        config.URL,
		Transport:  transport,
		Command:    config.Command,
		Args:       config.Args,
		AuthToken:  config.AuthToken,
		Enabled:   config.Enabled,
		Timeout:   timeout,
		MaxRetries: maxRetries,
		CreatedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// GetConfig returns the server configuration.
func (s *MCPServer) GetConfig() MCPServerConfig {
	return MCPServerConfig{
		Name:       s.Name,
		URL:        s.URL,
		Transport: s.Transport,
		Command:   s.Command,
		Args:      s.Args,
		AuthToken: s.AuthToken,
		Enabled:  s.Enabled,
		Timeout:  s.Timeout,
		MaxRetries: s.MaxRetries,
	}
}

// IsValid checks if the server configuration is valid.
func (s *MCPServer) IsValid() bool {
	if s.Name == "" || !s.Enabled {
		return false
	}
	if s.Transport == "http" && s.URL == "" {
		return false
	}
	if s.Transport == "stdio" && s.Command == "" {
		return false
	}
	return true
}

// Enable enables the server.
func (s *MCPServer) Enable() {
	s.Enabled = true
	s.UpdatedAt = time.Now()
}

// Disable disables the server.
func (s *MCPServer) Disable() {
	s.Enabled = false
	s.UpdatedAt = time.Now()
}

// SetAuthToken sets the authentication token securely.
func (s *MCPServer) SetAuthToken(token string) {
	s.AuthToken = token
	s.UpdatedAt = time.Now()
}
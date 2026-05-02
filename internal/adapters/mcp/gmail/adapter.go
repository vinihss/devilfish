package gmail

import (
	"context"
	"fmt"

	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// Email represents an email message from Gmail.
type Email struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// EmailCapability defines the interface for email operations.
// This interface is implemented by the Gmail MCP adapter.
type EmailCapability interface {
	// SendEmail sends an email via Gmail
	SendEmail(ctx context.Context, to, subject, body string) error

	// ListEmails lists emails matching the Gmail search query
	ListEmails(ctx context.Context, query string) ([]Email, error)

	// ReadEmail reads a specific email by ID
	ReadEmail(ctx context.Context, id string) (Email, error)

	// Name returns the provider name
	Name() string

	// Connect establishes connection to the Gmail MCP server
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// IsConnected returns connection status
	IsConnected() bool
}

// GmailMCP implements EmailCapability using Gmail MCP server.
type GmailMCP struct {
	config    outbound.MCPServerConfig
	client    outbound.MCPClient
	connected bool
	logger    *logging.ZLogger
}

// NewGmailMCP creates a new Gmail MCP adapter.
// The adapter uses the provided MCP client to communicate with the Gmail MCP server.
func NewGmailMCP(config outbound.MCPServerConfig, client outbound.MCPClient) *GmailMCP {
	logger := logging.NewLogger(nil).Named("gmail-mcp")

	return &GmailMCP{
		config:    config,
		client:    client,
		connected: false,
		logger:    logger,
	}
}

// Name returns the provider name.
func (g *GmailMCP) Name() string {
	return "gmail"
}

// SendEmail sends an email via the Gmail MCP server.
// It calls the "SendEmail" tool on the MCP server with the provided parameters.
func (g *GmailMCP) SendEmail(ctx context.Context, to, subject, body string) error {
	g.logger.Info("sending email via Gmail MCP")

	if !g.IsConnected() {
		return fmt.Errorf("gmail MCP: %w", ErrNotConnected)
	}

	if to == "" {
		return fmt.Errorf("gmail MCP: %w", ErrInvalidRecipient)
	}

	params := map[string]interface{}{
		"to":      to,
		"subject": subject,
		"body":    body,
	}

	_, err := g.client.ExecuteTool(ctx, "SendEmail", params)
	if err != nil {
		g.logger.Errorf("failed to send email via Gmail MCP: %v", err)
		return fmt.Errorf("gmail MCP: failed to send email: %w", err)
	}

	g.logger.Infof("email sent successfully via Gmail MCP to %s", to)
	return nil
}

// ListEmails lists emails matching the Gmail search query.
// It calls the "ListEmails" tool on the MCP server.
// The query parameter uses Gmail's search syntax (e.g., "is:unread", "from:someone@example.com").
func (g *GmailMCP) ListEmails(ctx context.Context, query string) ([]Email, error) {
	g.logger.Infof("listing emails via Gmail MCP with query: %s", query)

	if !g.IsConnected() {
		return nil, fmt.Errorf("gmail MCP: %w", ErrNotConnected)
	}

	params := map[string]interface{}{
		"query": query,
	}

	result, err := g.client.ExecuteTool(ctx, "ListEmails", params)
	if err != nil {
		g.logger.Errorf("failed to list emails via Gmail MCP: %v", err)
		return nil, fmt.Errorf("gmail MCP: failed to list emails: %w", err)
	}

	emails, err := parseEmailList(result)
	if err != nil {
		g.logger.Errorf("failed to parse email list: %v", err)
		return nil, fmt.Errorf("gmail MCP: failed to parse email list: %w", err)
	}

	g.logger.Infof("listed %d emails via Gmail MCP", len(emails))
	return emails, nil
}

// ReadEmail reads a specific email by ID.
// It calls the "ReadEmail" tool on the MCP server.
func (g *GmailMCP) ReadEmail(ctx context.Context, id string) (Email, error) {
	g.logger.Infof("reading email via Gmail MCP: %s", id)

	if !g.IsConnected() {
		return Email{}, fmt.Errorf("gmail MCP: %w", ErrNotConnected)
	}

	if id == "" {
		return Email{}, fmt.Errorf("gmail MCP: %w", ErrInvalidEmailID)
	}

	params := map[string]interface{}{
		"id": id,
	}

	result, err := g.client.ExecuteTool(ctx, "ReadEmail", params)
	if err != nil {
		g.logger.Errorf("failed to read email via Gmail MCP: %v", err)
		return Email{}, fmt.Errorf("gmail MCP: failed to read email: %w", err)
	}

	email, err := parseEmail(result)
	if err != nil {
		g.logger.Errorf("failed to parse email: %v", err)
		return Email{}, fmt.Errorf("gmail MCP: failed to parse email: %w", err)
	}

	g.logger.Infof("email read successfully via Gmail MCP: %s", id)
	return email, nil
}

// Connect establishes connection to the Gmail MCP server.
// It uses the configuration provided during adapter creation.
func (g *GmailMCP) Connect(ctx context.Context) error {
	g.logger.Infof("connecting to Gmail MCP server: %s", g.config.Name)

	if g.IsConnected() {
		g.logger.Info("already connected to Gmail MCP server")
		return nil
	}

	if g.client == nil {
		return fmt.Errorf("gmail MCP: %w", ErrNoClient)
	}

	if err := g.client.Connect(ctx, g.config); err != nil {
		g.logger.Errorf("failed to connect to Gmail MCP server: %v", err)
		return fmt.Errorf("gmail MCP: failed to connect: %w", err)
	}

	g.connected = true
	g.logger.Info("connected to Gmail MCP server successfully")
	return nil
}

// Disconnect closes the connection to the Gmail MCP server.
func (g *GmailMCP) Disconnect(ctx context.Context) error {
	g.logger.Info("disconnecting from Gmail MCP server")

	if !g.IsConnected() {
		g.logger.Info("not connected to Gmail MCP server")
		return nil
	}

	if g.client != nil {
		if err := g.client.Disconnect(ctx); err != nil {
			g.logger.Errorf("failed to disconnect from Gmail MCP server: %v", err)
			return fmt.Errorf("gmail MCP: failed to disconnect: %w", err)
		}
	}

	g.connected = false
	g.logger.Info("disconnected from Gmail MCP server")
	return nil
}

// IsConnected returns the connection status.
func (g *GmailMCP) IsConnected() bool {
	if g.client != nil {
		return g.client.IsConnected()
	}
	return g.connected
}

// parseEmailList parses a list of emails from MCP tool result.
func parseEmailList(result map[string]interface{}) ([]Email, error) {
	if result == nil {
		return []Email{}, nil
	}

	emailsRaw, ok := result["emails"]
	if !ok {
		// Try alternative key
		emailsRaw, ok = result["results"]
		if !ok {
			return []Email{}, nil
		}
	}

	emailsSlice, ok := emailsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid email list format")
	}

	emails := make([]Email, 0, len(emailsSlice))
	for _, raw := range emailsSlice {
		emailMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		email := Email{
			ID:      getStringValue(emailMap, "id"),
			From:    getStringValue(emailMap, "from"),
			To:      getStringValue(emailMap, "to"),
			Subject: getStringValue(emailMap, "subject"),
			Body:    getStringValue(emailMap, "body"),
		}
		emails = append(emails, email)
	}

	return emails, nil
}

// parseEmail parses a single email from MCP tool result.
func parseEmail(result map[string]interface{}) (Email, error) {
	if result == nil {
		return Email{}, fmt.Errorf("empty result")
	}

	// Check if result is wrapped in an "email" key
	emailRaw, ok := result["email"]
	if !ok {
		emailRaw = result
	}

	emailMap, ok := emailRaw.(map[string]interface{})
	if !ok {
		return Email{}, fmt.Errorf("invalid email format")
	}

	return Email{
		ID:      getStringValue(emailMap, "id"),
		From:    getStringValue(emailMap, "from"),
		To:      getStringValue(emailMap, "to"),
		Subject: getStringValue(emailMap, "subject"),
		Body:    getStringValue(emailMap, "body"),
	}, nil
}

// getStringValue safely extracts a string value from a map.
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Ensure GmailMCP implements EmailCapability interface.
var _ EmailCapability = (*GmailMCP)(nil)

package skill

import (
	"context"
	"fmt"

	"devilfish/internal/adapters/mcp"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
	// Gmail adapter is used via MCP registry, type assertion uses outbound.EmailCapability
)

// Gmail skill server name constant
const gmailServerName = "gmail"

// SendEmailSkill sends an email via Gmail MCP.
// It implements the Skill interface and provides a granular abstraction
// for the send_email capability.
type SendEmailSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewSendEmailSkill creates a new SendEmailSkill.
// The registry is used to resolve the Gmail MCP server.
func NewSendEmailSkill(registry *mcp.Registry) *SendEmailSkill {
	return &SendEmailSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.send_email"),
	}
}

// Name returns the unique identifier for this skill.
func (s *SendEmailSkill) Name() string {
	return "send_email"
}

// Description returns a human-readable description of what the skill does.
func (s *SendEmailSkill) Description() string {
	return "Send an email via Gmail. Requires 'to', 'subject', and 'body' parameters."
}

// Schema returns the JSON schema for the skill's input parameters.
// This schema is used by the LLM to understand what arguments are required.
func (s *SendEmailSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"to": map[string]interface{}{
				"type":        "string",
				"description": "Recipient email address",
			},
			"subject": map[string]interface{}{
				"type":        "string",
				"description": "Email subject line",
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Email body content",
			},
		},
		"required": []string{"to", "subject", "body"},
	}
}

// Execute sends an email via the Gmail MCP server.
// Required arguments: to, subject, body (all strings).
func (s *SendEmailSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing send_email skill")

	// 1. Validate input first
	to, ok := args["to"].(string)
	if !ok || to == "" {
		return "", fmt.Errorf("invalid or missing 'to' parameter: must be a non-empty string")
	}

	subject, ok := args["subject"].(string)
	if !ok {
		return "", fmt.Errorf("invalid 'subject' parameter: must be a string")
	}

	body, ok := args["body"].(string)
	if !ok {
		return "", fmt.Errorf("invalid 'body' parameter: must be a string")
	}

	// 2. Get Gmail MCP client from registry
	client, found := s.registry.Get(gmailServerName)
	if !found {
		return "", fmt.Errorf("gmail MCP server not found in registry")
	}

	// 3. Assert EmailCapability
	emailCap, ok := client.(outbound.EmailCapability)
	if !ok {
		// Try to get the adapter directly if registry stores it differently
		return "", fmt.Errorf("MCP client does not implement EmailCapability")
	}

	// 4. Call capability.SendEmail()
	s.logger.Infof("sending email to %s via Gmail MCP", to)
	err := emailCap.SendEmail(ctx, to, subject, body)
	if err != nil {
		s.logger.Errorf("failed to send email: %v", err)
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	// 5. Return success message
	result := fmt.Sprintf("Email sent successfully to %s", to)
	s.logger.Info(result)
	return result, nil
}

// ListEmailsSkill lists emails from Gmail.
// It implements the Skill interface and provides a granular abstraction
// for the list_emails capability.
type ListEmailsSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewListEmailsSkill creates a new ListEmailsSkill.
func NewListEmailsSkill(registry *mcp.Registry) *ListEmailsSkill {
	return &ListEmailsSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.list_emails"),
	}
}

// Name returns the unique identifier for this skill.
func (s *ListEmailsSkill) Name() string {
	return "list_emails"
}

// Description returns a human-readable description of what the skill does.
func (s *ListEmailsSkill) Description() string {
	return "List emails in Gmail matching a query. Uses Gmail search syntax (e.g., 'is:unread', 'from:example@gmail.com')."
}

// Schema returns the JSON schema for the skill's input parameters.
func (s *ListEmailsSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Gmail search query (e.g., 'is:unread', 'from:someone@example.com')",
			},
		},
		"required": []string{"query"},
	}
}

// Execute lists emails matching the query via the Gmail MCP server.
// Required arguments: query (string) - Gmail search syntax.
func (s *ListEmailsSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing list_emails skill")

	// 1. Validate input
	query, ok := args["query"].(string)
	if !ok || query == "" {
		// Use default query if not provided
		query = "in:inbox"
		s.logger.Infof("no query provided, using default: %s", query)
	}

	// 2. Get Gmail MCP client from registry
	client, found := s.registry.Get(gmailServerName)
	if !found {
		return "", fmt.Errorf("gmail MCP server not found in registry")
	}

	// 3. Assert EmailCapability
	emailCap, ok := client.(outbound.EmailCapability)
	if !ok {
		return "", fmt.Errorf("MCP client does not implement EmailCapability")
	}

	// 4. Call capability.ListEmails()
	s.logger.Infof("listing emails with query: %s", query)
	emails, err := emailCap.ListEmails(ctx, query)
	if err != nil {
		s.logger.Errorf("failed to list emails: %v", err)
		return "", fmt.Errorf("failed to list emails: %w", err)
	}

	// 5. Return structured output
	if len(emails) == 0 {
		return "No emails found matching the query.", nil
	}

	result := fmt.Sprintf("Found %d email(s):\n", len(emails))
	for i, email := range emails {
		if i >= 10 {
			result += fmt.Sprintf("... and %d more email(s)\n", len(emails)-10)
			break
		}
		result += fmt.Sprintf("- ID: %s | From: %s | Subject: %s\n", email.ID, email.From, email.Subject)
	}

	s.logger.Infof("listed %d emails", len(emails))
	return result, nil
}

// ReadEmailSkill reads a specific email from Gmail.
// It implements the Skill interface and provides a granular abstraction
// for the read_email capability.
type ReadEmailSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewReadEmailSkill creates a new ReadEmailSkill.
func NewReadEmailSkill(registry *mcp.Registry) *ReadEmailSkill {
	return &ReadEmailSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.read_email"),
	}
}

// Name returns the unique identifier for this skill.
func (s *ReadEmailSkill) Name() string {
	return "read_email"
}

// Description returns a human-readable description of what the skill does.
func (s *ReadEmailSkill) Description() string {
	return "Read a specific email by its ID. Returns the full email content including body."
}

// Schema returns the JSON schema for the skill's input parameters.
func (s *ReadEmailSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Email ID to read",
			},
		},
		"required": []string{"id"},
	}
}

// Execute reads an email by its ID via the Gmail MCP server.
// Required arguments: id (string) - the email ID.
func (s *ReadEmailSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing read_email skill")

	// 1. Validate input
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("invalid or missing 'id' parameter: must be a non-empty string")
	}

	// 2. Get Gmail MCP client from registry
	client, found := s.registry.Get(gmailServerName)
	if !found {
		return "", fmt.Errorf("gmail MCP server not found in registry")
	}

	// 3. Assert EmailCapability
	emailCap, ok := client.(outbound.EmailCapability)
	if !ok {
		return "", fmt.Errorf("MCP client does not implement EmailCapability")
	}

	// 4. Call capability.ReadEmail()
	s.logger.Infof("reading email with ID: %s", id)
	email, err := emailCap.ReadEmail(ctx, id)
	if err != nil {
		s.logger.Errorf("failed to read email: %v", err)
		return "", fmt.Errorf("failed to read email: %w", err)
	}

	// 5. Return structured output
	result := fmt.Sprintf("Email Details:\n")
	result += fmt.Sprintf("ID: %s\n", email.ID)
	result += fmt.Sprintf("From: %s\n", email.From)
	result += fmt.Sprintf("To: %s\n", email.To)
	result += fmt.Sprintf("Subject: %s\n", email.Subject)
	result += fmt.Sprintf("Body:\n%s\n", email.Body)

	s.logger.Infof("email read successfully: %s", id)
	return result, nil
}

// Ensure skill types implement the Skill interface
var _ Skill = (*SendEmailSkill)(nil)
var _ Skill = (*ListEmailsSkill)(nil)
var _ Skill = (*ReadEmailSkill)(nil)

package policy

import (
	"devilfish/internal/application/skill"
	"fmt"
	"strings"
)

// EmailPolicy prevents unsafe email operations.
// It can require confirmation for email sending and block specific recipients or domains.
type EmailPolicy struct {
	// RequireConfirmation makes send_email require explicit confirmation
	RequireConfirmation bool

	// AllowedDomains is a list of domains that emails can be sent to.
	// If empty, all domains are allowed (unless in BlockedRecipients).
	AllowedDomains []string

	// BlockedRecipients is a list of email addresses that are explicitly blocked.
	BlockedRecipients []string
}

// NewEmailPolicy creates a new EmailPolicy with default settings.
// By default, no confirmation is required and all recipients are allowed.
func NewEmailPolicy() *EmailPolicy {
	return &EmailPolicy{
		RequireConfirmation: false,
		AllowedDomains:     make([]string, 0),
		BlockedRecipients:  make([]string, 0),
	}
}

// Allow checks if an email-related tool call is allowed.
// Currently validates "send_email" tool calls.
func (p *EmailPolicy) Allow(call skill.ToolCall) error {
	// Only apply to email-related tools
	if call.Name != "send_email" {
		return nil
	}

	// Check if confirmation is required
	if p.RequireConfirmation {
		confirmed, ok := call.Parameters["confirmed"].(bool)
		if !ok || !confirmed {
			return fmt.Errorf("email sending requires confirmation (set 'confirmed: true')")
		}
	}

	// Get recipient from parameters
	recipient, err := getStringParam(call, "to")
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}

	// Check blocked recipients
	for _, blocked := range p.BlockedRecipients {
		if strings.EqualFold(recipient, blocked) {
			return fmt.Errorf("recipient %q is blocked", recipient)
		}
	}

	// Check allowed domains (if configured)
	if len(p.AllowedDomains) > 0 {
		domain := extractDomain(recipient)
		if domain == "" {
			return fmt.Errorf("cannot extract domain from recipient %q", recipient)
		}

		allowed := false
		for _, allowedDomain := range p.AllowedDomains {
			if strings.EqualFold(domain, allowedDomain) {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("domain %q is not in allowed domains list", domain)
		}
	}

	return nil
}

// getStringParam extracts a string parameter from the tool call.
func getStringParam(call skill.ToolCall, key string) (string, error) {
	val, ok := call.Parameters[key]
	if !ok {
		return "", fmt.Errorf("missing parameter %q", key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("parameter %q is not a string", key)
	}

	if str == "" {
		return "", fmt.Errorf("parameter %q cannot be empty", key)
	}

	return str, nil
}

// extractDomain extracts the domain part from an email address.
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

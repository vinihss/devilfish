package gmail

import "errors"

// Gmail MCP adapter errors.
var (
	// ErrNotConnected is returned when trying to perform operations while not connected.
	ErrNotConnected = errors.New("not connected to Gmail MCP server")

	// ErrInvalidRecipient is returned when the recipient email is empty.
	ErrInvalidRecipient = errors.New("invalid recipient email")

	// ErrInvalidEmailID is returned when the email ID is empty or invalid.
	ErrInvalidEmailID = errors.New("invalid email ID")

	// ErrNoClient is returned when the MCP client is not initialized.
	ErrNoClient = errors.New("MCP client not initialized")

	// ErrToolNotFound is returned when a required MCP tool is not found.
	ErrToolNotFound = errors.New("required Gmail tool not found on MCP server")

	// ErrInvalidResponse is returned when the MCP server returns an invalid response.
	ErrInvalidResponse = errors.New("invalid response from Gmail MCP server")
)

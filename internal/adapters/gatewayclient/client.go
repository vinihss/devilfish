// Package gatewayclient provides an HTTP client that implements inbound.MessageHandler
// by forwarding messages to the DevilFish core runtime (devilfishd).
//
// This is the boundary between the messaging runtime (messagingd) and the core
// runtime. The messaging adapters (Telegram, Discord, Slack) remain unchanged;
// they only see the inbound.MessageHandler interface.
package gatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"devilfish/internal/ports/inbound"
)

// Client forwards inbound messages to the core runtime over HTTP and returns
// the outbound response. It implements inbound.MessageHandler.
type Client struct {
	url    string
	apiKey string
	http   *http.Client
}

// Config holds the configuration for the gateway client.
type Config struct {
	// URL is the base URL of the core runtime, e.g. "http://localhost:8082".
	URL string
	// APIKey is an optional shared secret. When non-empty it is sent as
	// "Authorization: Bearer <api_key>" on every request.
	APIKey string
	// Timeout is the HTTP request timeout. Defaults to 30 s.
	Timeout time.Duration
}

// New creates a new Client with the given configuration.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		url:    cfg.URL,
		apiKey: cfg.APIKey,
		http:   &http.Client{Timeout: timeout},
	}
}

// Handle implements inbound.MessageHandler.
// It serialises the message as JSON, POSTs it to <gateway-url>/api/message,
// and deserialises the JSON response into an OutboundMessage.
func (c *Client) Handle(ctx context.Context, msg *inbound.InboundMessage) (*inbound.OutboundMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("inbound message cannot be nil")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/message", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to gateway: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gateway response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d: %s", resp.StatusCode, respBody)
	}

	var out inbound.OutboundMessage
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway response: %w", err)
	}

	return &out, nil
}

// Compile-time check that Client implements inbound.MessageHandler.
var _ inbound.MessageHandler = (*Client)(nil)

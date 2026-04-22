// Package slack provides a Slack bot adapter for DevilFish.
package slack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/infra/config"
	"devilfish/internal/ports/inbound"
)

// Adapter represents a Slack bot adapter.
// It handles incoming webhook events and sends messages back to users.
type Adapter struct {
	config     *config.SlackConfig
	handler    inbound.MessageHandler
	logger     zerolog.Logger
	client     *http.Client
	webhookURL string
	apiURL     string
	session    *Session
	mu         sync.RWMutex
}

// NewAdapter creates a new Slack bot adapter.
//
// The adapter can handle incoming webhook requests from Slack's API.
func NewAdapter(cfg config.SlackConfig, handler inbound.MessageHandler, logger zerolog.Logger) *Adapter {
	return &Adapter{
		config:     &cfg,
		handler:    handler,
		logger:     logger,
		client:     &http.Client{Timeout: 30 * time.Second},
		webhookURL: "https://hooks.slack.com/services",
		apiURL:     "https://slack.com/api",
	}
}

// Start starts the Slack adapter.
func (a *Adapter) Start(ctx context.Context) error {
	if !a.config.Enabled {
		a.logger.Info().Str("adapter", "slack").Msg("adapter disabled")
		return nil
	}

	a.logger.Info().Str("adapter", "slack").Msg("starting adapter")

	// Verify the bot token works
	if a.config.BotToken == "" {
		a.logger.Warn().Msg("bot token not configured")
	}

	// Start listening for events
	err := a.startEventSubscription(ctx)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to start event subscription")
		return err
	}

	a.logger.Info().Str("adapter", "slack").Msg("adapter started")
	return nil
}

// startEventSubscription starts an event subscription for the bot.
func (a *Adapter) startEventSubscription(ctx context.Context) error {
	// Verify auth with Slack
	if a.config.BotToken != "" {
		authTest, err := a.authTest(ctx)
		if err != nil {
			a.logger.Warn().Err(err).Msg("failed to verify bot auth")
		} else {
			a.logger.Info().Str("team", authTest.TeamID).Str("bot", authTest.BotID).Msg("bot authenticated")
		}
	}

	// The event subscription is handled via HTTP webhook
	return nil
}

// authTest tests the bot authentication.
func (a *Adapter) authTest(ctx context.Context) (*AuthTestResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", a.apiURL+"/auth.test", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.config.BotToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AuthTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("auth test failed: %s", result.Error)
	}

	return &result, nil
}

// HandleWebhook handles incoming webhook requests from Slack.
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify request method
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data first (for slash commands)
	if err := r.ParseForm(); err != nil {
		a.logger.Error().Err(err).Msg("failed to parse form")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Handle URL verification challenge (for event subscriptions)
	challenge := r.FormValue("challenge")
	if challenge != "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	// Handle slash command
	command := r.FormValue("command")
	if command != "" {
		a.handleSlashCommand(w, r)
		return
	}

	// Handle interactive component (buttons, menus)
	payload := r.FormValue("payload")
	if payload != "" {
		a.handleInteractive(w, payload)
		return
	}

	// Handle event payload (from Events API)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to read request body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse as event payload
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		a.logger.Error().Err(err).Msg("failed to unmarshal event")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process the event in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a.processEvent(ctx, event)
	}()

	w.WriteHeader(http.StatusOK)
}

// handleSlashCommand handles incoming slash commands from Slack.
func (a *Adapter) handleSlashCommand(w http.ResponseWriter, r *http.Request) {
	command := r.FormValue("command")
	text := r.FormValue("text")
	userID := r.FormValue("user_id")
	channelID := r.FormValue("channel_id")
	responseURL := r.FormValue("response_url")

	a.logger.Info().
		Str("command", command).
		Str("user_id", userID).
		Str("channel_id", channelID).
		Msg("slash command received")

	// Skip if no content
	if text == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Process command in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		inboundMsg := &inbound.InboundMessage{
			ID:      fmt.Sprintf("%d", time.Now().Unix()),
			UserID:  userID,
			Channel: "slack",
			Type:    "text",
			Content: text,
			Meta: map[string]any{
				"command":      command,
				"channel_id":   channelID,
				"response_url": responseURL,
			},
			Timestamp: time.Now(),
		}

		response, err := a.handler.Handle(ctx, inboundMsg)
		if err != nil {
			a.logger.Error().Err(err).Msg("failed to handle command")
			a.postMessage(ctx, channelID, "Sorry, I couldn't process your command.")
			return
		}

		if response != nil && response.Content != "" {
			a.postMessage(ctx, channelID, response.Content)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// handleInteractive handles interactive components (buttons, menus).
func (a *Adapter) handleInteractive(w http.ResponseWriter, payload string) {
	var callback Callback
	if err := json.Unmarshal([]byte(payload), &callback); err != nil {
		a.logger.Error().Err(err).Msg("failed to unmarshal callback")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process callback in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		inboundMsg := &inbound.InboundMessage{
			ID:      callback.MessageTs,
			UserID:  callback.User.ID,
			Channel: callback.Channel.ID,
			Type:    "text",
			Content: callback.ActionData.CustomID,
			Meta: map[string]any{
				"callback_id": callback.CallbackID,
				"action_name": callback.ActionData.Name,
			},
			Timestamp: time.Now(),
		}

		response, err := a.handler.Handle(ctx, inboundMsg)
		if err != nil {
			a.logger.Error().Err(err).Msg("failed to handle callback")
			a.postMessage(ctx, callback.Channel.ID, "Sorry, I couldn't process your request.")
			return
		}

		if response != nil && response.Content != "" {
			// Update the message with response
			a.updateMessage(ctx, callback.MessageTs, callback.Channel.ID, response.Content)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// updateMessage updates an existing Slack message.
func (a *Adapter) updateMessage(ctx context.Context, ts, channelID, text string) error {
	if text == "" {
		return nil
	}

	// Slack has a 3000 character limit per message
	if len(text) > 3000 {
		text = text[:2997] + "..."
	}

	payload := map[string]any{
		"ts":      ts,
		"channel": channelID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		a.apiURL+"/chat.update",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// Callback represents a Slack interactive callback.
type Callback struct {
	CallbackID string `json:"callback_id"`
	User       struct {
		ID string `json:"id"`
	} `json:"user"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
	MessageTs  string `json:"message_ts"`
	ActionData struct {
		Name     string `json:"name"`
		CustomID string `json:"custom_id"`
	} `json:"actions"`
}

// VerifySignature verifies the Slack request signature.
func (a *Adapter) VerifySignature(signature string, timestamp string, body []byte) bool {
	if a.config.BotToken == "" {
		return true // Skip verification if no token
	}

	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	mac := hmac.New(sha256.New, []byte(a.config.BotToken))
	mac.Write([]byte(baseString))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// processEvent processes an incoming Slack event.
func (a *Adapter) processEvent(ctx context.Context, event Event) {
	// Check allowlist
	if len(a.config.AllowList) > 0 && event.ChannelID != "" {
		allowed := false
		for _, channelID := range a.config.AllowList {
			if channelID == event.ChannelID {
				allowed = true
				break
			}
		}
		if !allowed {
			a.logger.Debug().Str("channel_id", event.ChannelID).Msg("channel not in allowlist")
			return
		}
	}

	// Determine event type and content
	content := event.Text
	if content == "" && event.Message != nil {
		content = event.Message.Text
	}

	if content == "" {
		return
	}

	// Remove bot mention from text
	content = strings.TrimPrefix(content, "<@"+event.BotID+">")
	content = strings.TrimSpace(content)

	if content == "" {
		return
	}

	// Create inbound message
	inboundMsg := &inbound.InboundMessage{
		ID:      event.EventTime,
		UserID:  event.UserID,
		Channel: "slack",
		Type:    "text",
		Content: content,
		Meta: map[string]any{
			"channel_id":   event.ChannelID,
			"team_id":      event.TeamID,
			"event_time":   event.EventTime,
			"thread_ts":    event.ThreadTimeStamp,
			"channel_name": event.ChannelName,
		},
		Timestamp: time.Now(),
	}

	// Handle message
	response, err := a.handler.Handle(ctx, inboundMsg)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to handle message")
		a.postMessage(ctx, event.ChannelID, "Sorry, I couldn't process your message.")
		return
	}

	// Send response
	if response != nil && response.Content != "" {
		a.postMessage(ctx, event.ChannelID, response.Content)
	}
}

// Send implements the outbound.MessageSource interface.
func (a *Adapter) Send(ctx context.Context, channel string, to string, content string) error {
	return a.postMessage(ctx, channel, content)
}

// postMessage posts a message to a Slack channel.
func (a *Adapter) postMessage(ctx context.Context, channelID string, text string) error {
	if text == "" {
		return nil
	}

	// Slack has a 3000 character limit per message
	if len(text) > 3000 {
		text = text[:2997] + "..."
	}

	payload := map[string]any{
		"channel": channelID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		a.apiURL+"/chat.postMessage",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API error: %s", body)
	}

	return nil
}

// Shutdown gracefully shuts down the adapter.
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.session != nil {
		a.session.Close()
	}
	return nil
}

// Session represents a Slack bot session.
type Session struct {
	teamID string
	botID  string
	closed bool
	mu     sync.Mutex
}

// Close closes the session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// Event represents a Slack event.
type Event struct {
	Type            string   `json:"type"`
	ChannelID       string   `json:"channel_id"`
	ChannelName     string   `json:"channel_name"`
	UserID          string   `json:"user_id"`
	Text            string   `json:"text"`
	TeamID          string   `json:"team_id"`
	BotID           string   `json:"bot_id"`
	EventTime       string   `json:"event_time"`
	ThreadTimeStamp string   `json:"thread_ts"`
	Message         *Message `json:"message"`
}

// Message represents a Slack message.
type Message struct {
	Type       string `json:"type"`
	SubType    string `json:"subtype"`
	Channel    string `json:"channel"`
	User       string `json:"user"`
	Text       string `json:"text"`
	Timestamp  string `json:"ts"`
	ThreadTime string `json:"thread_ts"`
	BotID      string `json:"bot_id"`
}

// AuthTestResponse represents the auth.test API response.
type AuthTestResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error"`
	URL    string `json:"url"`
	Team   string `json:"team"`
	TeamID string `json:"team_id"`
	User   string `json:"user"`
	BotID  string `json:"bot_id"`
}

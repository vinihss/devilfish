// Package telegram provides a Telegram bot adapter for DevilFish.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/infra/config"
	"devilfish/internal/ports/inbound"
)

// Adapter represents a Telegram bot adapter.
// It handles incoming webhook events and sends messages back to users.
type Adapter struct {
	config  *config.TelegramConfig
	handler inbound.MessageHandler
	logger  zerolog.Logger
	client  *http.Client
	botURL  string
	httpSrv *http.Server
}

// NewAdapter creates a new Telegram bot adapter.
//
// The adapter can handle incoming webhook requests from Telegram's Bot API.
func NewAdapter(cfg config.TelegramConfig, handler inbound.MessageHandler, logger zerolog.Logger) *Adapter {
	botURL := ""
	if cfg.BotToken != "" {
		botURL = "https://api.telegram.org/bot" + cfg.BotToken
	}

	return &Adapter{
		config:  &cfg,
		handler: handler,
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
		botURL:  botURL,
	}
}

// Start starts the adapter's webhook server.
// It listens for incoming updates from Telegram and processes them.
func (a *Adapter) Start(ctx context.Context) error {
	if !a.config.Enabled {
		a.logger.Info().Str("adapter", "telegram").Msg("adapter disabled")
		return nil
	}

	if a.botURL == "" {
		a.logger.Warn().Msg("no bot token configured, skipping polling")
		return nil
	}

	a.logger.Info().Str("adapter", "telegram").Msg("starting adapter")

	// Note: In production, you would configure the actual webhook URL
	// via Telegram's setWebhook API. For development, you might use polling.
	go func() {
		if err := a.startPolling(ctx); err != nil {
			a.logger.Error().Err(err).Msg("polling error")
		}
	}()

	a.logger.Info().Str("adapter", "telegram").Msg("adapter started")
	return nil
}

// startPolling starts polling for updates (development mode).
func (a *Adapter) startPolling(ctx context.Context) error {
	offset := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	a.logger.Info().Msg("polling loop started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("polling loop stopped by context")
			return nil // Don't return error on shutdown
		case <-ticker.C:
			updates, err := a.getUpdates(offset)
			if err != nil {
				a.logger.Warn().Err(err).Msg("getUpdates error, retrying...")
				continue
			}

			for _, update := range updates {
				if update.UpdateID >= offset {
					offset = update.UpdateID + 1
				}

				if update.Message == nil || update.Message.Text == "" {
					continue
				}

				a.logger.Info().Int("update_id", update.UpdateID).Str("text", update.Message.Text).Msg("received message")
				go a.processMessage(context.Background(), update.Message)
			}
		}
	}
}

// getUpdates retrieves updates from Telegram.
func (a *Adapter) getUpdates(offset int) ([]Update, error) {
	// Use short timeout to avoid long blocking
	url := fmt.Sprintf("%s/getUpdates?timeout=5&offset=%d", a.botURL, offset)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", body)
	}

	return result.Result, nil
}

// HandleWebhook handles incoming webhook requests from Telegram.
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to read request body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		a.logger.Error().Err(err).Msg("failed to unmarshal update")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process message in background
	go func() {
		if update.Message != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			a.processMessage(ctx, update.Message)
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

// processMessage processes an incoming Telegram message.
func (a *Adapter) processMessage(ctx context.Context, msg *Message) {
	// Check allowlist
	if len(a.config.AllowList) > 0 {
		allowed := false
		for _, userID := range a.config.AllowList {
			if userID == fmt.Sprintf("%d", msg.From.ID) {
				allowed = true
				break
			}
		}
		if !allowed {
			a.logger.Debug().Str("user_id", fmt.Sprintf("%d", msg.From.ID)).Msg("user not in allowlist")
			return
		}
	}

	// Convert to inbound message
	inboundMsg := &inbound.InboundMessage{
		ID:        fmt.Sprintf("%d", msg.MessageID),
		UserID:    fmt.Sprintf("%d", msg.From.ID),
		Channel:   "telegram",
		Type:      "text",
		Content:   msg.Text,
		Meta:      map[string]any{"chat_id": fmt.Sprintf("%d", msg.Chat.ID)},
		Timestamp: time.Unix(int64(msg.Date), 0),
	}

	// Handle message
	response, err := a.handler.Handle(ctx, inboundMsg)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to handle message")
		a.sendMessage(ctx, fmt.Sprintf("%d", msg.Chat.ID), "Sorry, I couldn't process your message.")
		return
	}

	// Send response
	if response != nil && response.Content != "" {
		a.sendMessage(ctx, fmt.Sprintf("%d", msg.Chat.ID), response.Content)
	}
}

// Send implements the outbound.MessageSource interface.
func (a *Adapter) Send(ctx context.Context, channel string, to string, content string) error {
	return a.sendMessage(ctx, to, content)
}

// sendMessage sends a text message to a specific chat.
func (a *Adapter) sendMessage(ctx context.Context, chatID string, text string) error {
	if text == "" {
		return nil
	}

	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.botURL+"/sendMessage",
		strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", body)
	}

	return nil
}

// Shutdown gracefully shuts down the adapter.
func (a *Adapter) Shutdown(ctx context.Context) error {
	if a.httpSrv != nil {
		return a.httpSrv.Shutdown(ctx)
	}
	return nil
}

// Update represents a Telegram update.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"`
	Date      int    `json:"date"`
}

// User represents a Telegram user.
type User struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

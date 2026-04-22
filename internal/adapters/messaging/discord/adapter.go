// Package discord provides a Discord bot adapter for DevilFish.
package discord

import (
	"context"
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

// Adapter represents a Discord bot adapter.
// It handles incoming webhook events and sends messages back to users.
type Adapter struct {
	config  *config.DiscordConfig
	handler inbound.MessageHandler
	logger  zerolog.Logger
	client  *http.Client
	botURL  string
	session *Session
	mu      sync.RWMutex
}

// NewAdapter creates a new Discord bot adapter.
//
// The adapter can handle incoming webhook requests from Discord's API.
func NewAdapter(cfg config.DiscordConfig, handler inbound.MessageHandler, logger zerolog.Logger) *Adapter {
	a := &Adapter{
		config:  &cfg,
		handler: handler,
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
		botURL:  "https://discord.com/api/v10",
	}
	return a
}

// Start starts the Discord adapter.
func (a *Adapter) Start(ctx context.Context) error {
	if !a.config.Enabled {
		a.logger.Info().Str("adapter", "discord").Msg("adapter disabled")
		return nil
	}

	a.logger.Info().Str("adapter", "discord").Msg("starting adapter")

	// Set up the bot session with Discord gateway
	// In production, you would use a proper WebSocket connection
	session, err := a.startSession(ctx)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to start session")
		return err
	}

	a.mu.Lock()
	a.session = session
	a.mu.Unlock()

	a.logger.Info().Str("adapter", "discord").Msg("adapter started")
	return nil
}

// startSession starts a Discord bot session.
func (a *Adapter) startSession(ctx context.Context) (*Session, error) {
	// For webhook-based mode, we don't need a gateway session
	// Just verify the bot token works
	if a.config.BotToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	// Verify the bot
	me, err := a.getCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify bot: %w", err)
	}

	a.logger.Info().Str("bot", me.Username).Msg("bot verified")

	return &Session{
		user: me,
	}, nil
}

// getCurrentUser gets the current bot user.
func (a *Adapter) getCurrentUser(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.botURL+"/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+a.config.BotToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API error: %s", body)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// HandleWebhook handles incoming webhook requests from Discord.
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify the request is from Discord using the interaction endpoint URL
	// In production, you'd verify the signature

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to read request body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the interaction
	var interaction Interaction
	if err := json.Unmarshal(body, &interaction); err != nil {
		a.logger.Error().Err(err).Msg("failed to unmarshal interaction")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Handle different interaction types
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		switch interaction.Type {
		case InteractionTypeMessageComponent:
			a.handleMessageComponent(ctx, interaction)
		case InteractionTypePing:
			a.handlePing(w, interaction)
		default:
			// For regular messages, we would need to use the gateway
			a.logger.Debug().Int("type", interaction.Type).Msg("unhandled interaction type")
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

// handlePing responds to Discord ping interactions.
func (a *Adapter) handlePing(w http.ResponseWriter, interaction Interaction) {
	// Respond to ping with pong
	response := map[string]int{
		"type": InteractionResponseTypePong,
	}
	body, _ := json.Marshal(response)
	w.Write(body)
}

// handleMessageComponent handles message component interactions (buttons, selects).
func (a *Adapter) handleMessageComponent(ctx context.Context, interaction Interaction) {
	if interaction.Message == nil {
		return
	}

	// Determine user and channel from the interaction
	userID := ""
	if interaction.Member != nil {
		userID = interaction.Member.User.ID
	} else if interaction.User != nil {
		userID = interaction.User.ID
	}

	channelID := interaction.ChannelID
	if channelID == "" && interaction.Message != nil {
		channelID = interaction.Message.ChannelID
	}

	// Extract content from the component
	content := ""
	if interaction.Data != nil {
		content = interaction.Data.CustomID
	}

	// Create inbound message
	inboundMsg := &inbound.InboundMessage{
		ID:      interaction.Message.ID,
		UserID:  userID,
		Channel: "discord",
		Type:    "text",
		Content: content,
		Meta: map[string]any{
			"channel_id": channelID,
			"guild_id":   interaction.GuildID,
		},
		Timestamp: time.Now(),
	}

	// Handle message
	response, err := a.handler.Handle(ctx, inboundMsg)
	if err != nil {
		a.logger.Error().Err(err).Msg("failed to handle message")
		a.sendMessage(ctx, channelID, "Sorry, I couldn't process your request.")
		return
	}

	// Send response as followup message
	if response != nil && response.Content != "" {
		a.sendMessage(ctx, channelID, response.Content)
	}
}

// Send implements the outbound.MessageSource interface.
func (a *Adapter) Send(ctx context.Context, channel string, to string, content string) error {
	return a.sendMessage(ctx, channel, content)
}

// sendMessage sends a message to a Discord channel.
func (a *Adapter) sendMessage(ctx context.Context, channelID string, content string) error {
	if content == "" {
		return nil
	}

	// Discord has a 2000 character limit per message
	if len(content) > 2000 {
		content = content[:1997] + "..."
	}

	payload := map[string]any{
		"content": content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		a.botURL+"/channels/"+channelID+"/messages",
		strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+a.config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error: %s", body)
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

// Session represents a Discord bot session.
type Session struct {
	user   *User
	closed bool
}

// Close closes the session.
func (s *Session) Close() {
	s.closed = true
}

// User represents a Discord user.
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Bot           bool   `json:"bot"`
}

// Member represents a Discord guild member.
type Member struct {
	User     *User    `json:"user"`
	Nick     string   `json:"nick"`
	Roles    []string `json:"roles"`
	JoinedAt string   `json:"joined_at"`
}

// Interaction represents a Discord interaction.
type Interaction struct {
	ID            string           `json:"id"`
	ApplicationID string           `json:"application_id"`
	Type          int              `json:"type"`
	Data          *InteractionData `json:"data"`
	GuildID       string           `json:"guild_id"`
	ChannelID     string           `json:"channel_id"`
	Message       *Message         `json:"message"`
	Member        *Member          `json:"member"`
	User          *User            `json:"user"`
	Token         string           `json:"token"`
}

// InteractionData represents the data of an interaction.
type InteractionData struct {
	CustomID string `json:"custom_id"`
	Type     int    `json:"type"`
}

// Message represents a Discord message.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Author    *User  `json:"author"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// Constants
const (
	InteractionTypePing             int = 1
	InteractionTypeMessageComponent int = 3

	InteractionResponseTypePong int = 1
)

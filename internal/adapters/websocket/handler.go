package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"devilfish/internal/ports/inbound"
)

// Handler handles WebSocket connections.
// It implements http.Handler interface for integration with net/http.
type Handler struct {
	upgrader websocket.Upgrader
	handler  inbound.MessageHandler
	logger   zerolog.Logger
	config   Config
}

// Config holds the WebSocket handler configuration.
type Config struct {
	Host        string
	Port        int
	AuthEnabled bool
	JWTSecret   string
}

// NewHandler creates a new WebSocket handler.
func NewHandler(
	handler inbound.MessageHandler,
	logger zerolog.Logger,
	config Config,
) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		handler: handler,
		logger:  logger,
		config:  config,
	}
}

// HandleHTTP handles HTTP requests and upgrades to WebSocket.
func (h *Handler) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// Currently, we skip JWT validation for simplicity
	// In production, add JWT validation here

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade connection")
		return
	}
	defer conn.Close()

	// Handle the connection
	h.handleConnection(conn)
}

// handleConnection handles a WebSocket connection.
func (h *Handler) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	// Configure connection
	conn.SetReadLimit(8192)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Write pump in goroutine
	go h.writePump(conn)

	// Read pump
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warn().Err(err).Msg("connection error")
			}
			return
		}

		// Process message
		if err := h.processMessage(conn, data); err != nil {
			h.logger.Warn().Err(err).Msg("failed to process message")
			// Send error response
			h.sendError(conn, err.Error())
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (h *Handler) writePump(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processMessage processes an incoming WebSocket message.
func (h *Handler) processMessage(conn *websocket.Conn, data []byte) error {
	// Parse inbound message (simple JSON parsing)
	// In production, use proper JSON parsing

	var inboundMsg inbound.InboundMessage
	// For now, assume the message is just a string content
	// wrapped in a JSON object with "content" field
	inboundMsg.Content = string(data)
	inboundMsg.Channel = "ws"
	inboundMsg.Type = "text"

	// Process message via handler
	ctx := context.Background()
	outboundMsg, err := h.handler.Handle(ctx, &inboundMsg)
	if err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}

	// Send response if present
	if outboundMsg != nil {
		return conn.WriteJSON(outboundMsg)
	}

	return nil
}

// sendError sends an error message to the client.
func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	msg := inbound.OutboundMessage{
		Channel: "ws",
		Type:    "error",
		Content: errMsg,
	}
	_ = conn.WriteJSON(msg)
}

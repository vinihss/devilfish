package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"devilfish/internal/ports/inbound"
)

// constants for WebSocket configuration.
const (
	// pingWriteInterval is the interval for writing ping frames.
	pingWriteInterval = 30 * time.Second

	// pongReadTimeout is the timeout for receiving pong frames.
	pongReadTimeout = 60 * time.Second

	// maxMessageSize is the maximum message size in bytes.
	maxMessageSize = 8192
)

// upgrader upgrades HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development; configure properly in production
	},
}

// gateway implements the WebSocketHandler interface.
type gateway struct {
	server       *http.Server
	upgrader     websocket.Upgrader
	handler      inbound.MessageHandler
	jwtValidator JWTValidator
	logger       zerolog.Logger
	config       ServerConfig
	clients      map[string]*client
	clientsMu    sync.RWMutex
	pingInterval time.Duration
	pongTimeout  time.Duration
}

// ServerConfig holds the WebSocket server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// JWTValidator validates JWT tokens.
type JWTValidator interface {
	ValidateToken(token string) (string, error) // Returns userID or error
}

// client represents a connected WebSocket client.
type client struct {
	conn    *websocket.Conn
	userID  string
	sendMsg chan []byte
	done    chan struct{}
}

// NewGateway creates a new WebSocket gateway.
//
// Parameters:
//   - handler: Message handler for processing incoming messages
//   - jwtValidator: JWT validator for authenticating connections
//   - logger: Logger for logging messages
//   - config: Server configuration (host, port, timeouts)
//
// Returns a new WebSocket gateway.
func NewGateway(
	handler inbound.MessageHandler,
	jwtValidator JWTValidator,
	logger zerolog.Logger,
	config ServerConfig,
) *gateway {
	return &gateway{
		handler:      handler,
		jwtValidator: jwtValidator,
		logger:       logger,
		config:       config,
		clients:      make(map[string]*client),
		pingInterval: pingWriteInterval,
		pongTimeout:  pongReadTimeout,
		upgrader:     upgrader,
	}
}

// Start starts the WebSocket gateway server.
//
// This method blocks until the server is stopped.
// Returns an error if the server fails to start.
func (g *gateway) Start() error {
	g.logger.Info().Str("host", g.config.Host).Int("port", g.config.Port).Msg("starting WebSocket gateway")

	addr := fmt.Sprintf("%s:%d", g.config.Host, g.config.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", g.handleHTTP)

	g.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  g.config.ReadTimeout,
		WriteTimeout: g.config.WriteTimeout,
	}

	if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start WebSocket server: %w", err)
	}

	return nil
}

// Stop stops the WebSocket gateway server.
//
// It gracefully shuts down all client connections and stops the server.
func (g *gateway) Stop(ctx context.Context) error {
	g.logger.Info().Msg("stopping WebSocket gateway")

	// Close all client connections
	g.clientsMu.RLock()
	for _, c := range g.clients {
		close(c.done)
		c.conn.Close()
	}
	g.clientsMu.RUnlock()

	// Shutdown server
	if g.server != nil {
		if err := g.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown WebSocket server: %w", err)
		}
	}

	return nil
}

// handleHTTP handles incoming HTTP requests and upgrades them to WebSocket.
func (g *gateway) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract JWT token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}

	// Validate JWT token
	userID, err := g.jwtValidator.ValidateToken(token)
	if err != nil {
		g.logger.Warn().Err(err).Msg("invalid token")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Warn().Err(err).Msg("failed to upgrade connection")
		return
	}

	// Create client
	client := &client{
		conn:    conn,
		userID:  userID,
		sendMsg: make(chan []byte, 256),
		done:    make(chan struct{}),
	}

	// Register client
	g.registerClient(client)

	// Handle connection in goroutine
	go g.handleConnection(client)
}

// handleConnection handles a WebSocket connection.
//
// It reads messages from the client and processes them.
// Handles ping/pong for keepalive and graceful disconnection.
func (g *gateway) handleConnection(client *client) {
	defer func() {
		g.unregisterClient(client)
		close(client.sendMsg)
		client.conn.Close()
		g.logger.Info().Str("user_id", client.userID).Msg("client disconnected")
	}()

	// Configure connection
	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(g.pongTimeout))
	client.conn.SetPongHandler(func(appData string) error {
		client.conn.SetReadDeadline(time.Now().Add(g.pongTimeout))
		return nil
	})

	// Goroutine for sending messages
	go func() {
		ticker := time.NewTicker(g.pingInterval)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-client.sendMsg:
				if !ok {
					return
				}
				client.conn.SetWriteDeadline(time.Now().Add(g.config.WriteTimeout))
				if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					g.logger.Warn().Err(err).Msg("failed to write message")
					return
				}

			case <-ticker.C:
				client.conn.SetWriteDeadline(time.Now().Add(g.config.WriteTimeout))
				if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}

			case <-client.done:
				return
			}
		}
	}()

	g.logger.Info().Str("user_id", client.userID).Msg("client connected")

	// Read loop
	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				g.logger.Warn().Err(err).Msg("unexpected close error")
			}
			return
		}

		// Process message
		if err := g.processMessage(client, data); err != nil {
			g.logger.Warn().Err(err).Msg("failed to process message")
			// Send error response
			g.sendError(client, err.Error())
		}
	}
}

// processMessage processes an incoming WebSocket message.
func (g *gateway) processMessage(client *client, data []byte) error {
	// Parse inbound message
	var inboundMsg inbound.InboundMessage
	if err := json.Unmarshal(data, &inboundMsg); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Set user ID from connection
	inboundMsg.UserID = client.userID

	// Process message via handler
	ctx := context.Background()
	outboundMsg, err := g.handler.Handle(ctx, &inboundMsg)
	if err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}

	// Send response if present
	if outboundMsg != nil {
		return g.sendToClient(client, outboundMsg)
	}

	return nil
}

// sendToClient sends a message to a specific client.
func (g *gateway) sendToClient(client *client, msg *inbound.OutboundMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case client.sendMsg <- data:
		return nil
	case <-client.done:
		return fmt.Errorf("connection closed")
	}
}

// sendError sends an error message to a client.
func (g *gateway) sendError(client *client, errMsg string) {
	msg := inbound.OutboundMessage{
		To:      client.userID,
		Channel: "ws",
		Type:    "error",
		Content: errMsg,
	}
	_ = g.sendToClient(client, &msg)
}

// registerClient registers a client.
func (g *gateway) registerClient(client *client) {
	g.clientsMu.Lock()
	defer g.clientsMu.Unlock()
	g.clients[client.userID] = client
}

// unregisterClient unregisters a client.
func (g *gateway) unregisterClient(client *client) {
	g.clientsMu.Lock()
	defer g.clientsMu.Unlock()
	delete(g.clients, client.userID)
}

// HandleConnection implements the WebSocketHandler interface for a raw WebSocket connection.
// This is used when upgrading an existing WebSocket connection.
func (g *gateway) HandleConnection(ctx context.Context, conn inbound.WebSocketConnection) error {
	// Note: This method is for handling pre-upgraded WebSocket connections
	// The actual implementation uses HTTP upgrade flow
	return g.handleRawConnection(ctx, conn)
}

// Broadcast implements the WebSocketHandler interface.
// Sends a message to all connected clients.
func (g *gateway) Broadcast(ctx context.Context, msg *inbound.OutboundMessage) error {
	g.clientsMu.RLock()
	defer g.clientsMu.RUnlock()

	if len(g.clients) == 0 {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(g.clients))

	for _, c := range g.clients {
		client := c // Create a new variable to capture the loop variable
		go func() {
			defer wg.Done()
			select {
			case client.sendMsg <- data:
			case <-client.done:
			case <-ctx.Done():
			}
		}()
	}

	wg.Wait()
	return nil
}

// handleRawConnection handles a raw WebSocket connection.
func (g *gateway) handleRawConnection(ctx context.Context, conn inbound.WebSocketConnection) error {
	// For extended conn support - placeholder for future extension
	return nil
}

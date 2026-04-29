package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/adapters/ai/groq"
	"devilfish/internal/adapters/ai/openai"
	"devilfish/internal/adapters/messaging"
	"devilfish/internal/adapters/storage/memory"
	"devilfish/internal/adapters/websocket"
	"devilfish/internal/application/usecase"
	"devilfish/internal/infra/config"
	"devilfish/internal/infra/i18n"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/inbound"
	"devilfish/internal/ports/outbound"
)

func main() {
	// Initialize logger
	zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger := logging.NewLoggerAdapter(&zerologLogger)
	logger.Info("starting DevilFish")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Warn(fmt.Sprintf("using default config: %v", err))
		cfg = config.DefaultConfig()
	}

	logger.Info(fmt.Sprintf("server on %s:%d", cfg.Server.Host, cfg.Server.Port))
	logger.Info(fmt.Sprintf("configured providers: %v", func() []string {
		r := []string{}
		for _, p := range cfg.AI.Providers {
			if p.Enabled {
				r = append(r, fmt.Sprintf("%s(api_key_set=%v)", p.Name, p.APIKey != ""))
			}
		}
		return r
	}()))

	// Load translator
	translator, err := i18n.LoadTranslator(
		"./internal/infra/i18n/locales",
		cfg.I18n.DefaultLocale,
		cfg.I18n.FallbackLocale,
	)
	if err != nil {
		logger.Warn("using fallback translator")
		translator = i18n.NewTranslator(nil)
	}

	// Create AI providers
	aiProviders := make(map[string]outbound.AIProvider)
	for _, providerCfg := range cfg.AI.Providers {
		if !providerCfg.Enabled {
			continue
		}
		if providerCfg.APIKey == "" {
			logger.Warn(fmt.Sprintf("provider %s enabled but no API key", providerCfg.Name))
			continue
		}

		var provider outbound.AIProvider
		var err error
		switch providerCfg.Name {
		case "openai":
			provider, err = openai.NewProvider(providerCfg, &zerologLogger)
		case "groq":
			provider, err = groq.NewProvider(providerCfg, &zerologLogger)
		case "gemini":
			// TODO: implement gemini provider
			logger.Warn("gemini provider not implemented yet")
		case "ollama":
			// TODO: implement ollama provider
			logger.Warn("ollama provider not implemented yet")
		default:
			logger.Warn(fmt.Sprintf("unknown provider: %s", providerCfg.Name))
			continue
		}

		if err != nil {
			logger.Error(fmt.Sprintf("failed to create %s: %v", providerCfg.Name, err))
			continue
		}

		if provider == nil {
			logger.Warn(fmt.Sprintf("provider %s returned nil", providerCfg.Name))
			continue
		}

		logger.Info(fmt.Sprintf("created provider: %s", providerCfg.Name))
		aiProviders[providerCfg.Name] = provider
	}

	logger.Info(fmt.Sprintf("provider map: %v", aiProviders))

	// Check default provider
	logger.Info(fmt.Sprintf("looking for: '%s'", cfg.AI.DefaultProvider))
	logger.Info(fmt.Sprintf("keys in map: %v", func() []string {
		r := []string{}
		for k := range aiProviders {
			r = append(r, k)
		}
		return r
	}()))
	defaultProvider, ok := aiProviders[cfg.AI.DefaultProvider]
	logger.Info(fmt.Sprintf("ok=%v, provider=%v", ok, defaultProvider))
	if !ok {
		logger.Error("no valid AI provider found")
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("using provider: %s", cfg.AI.DefaultProvider))

	// Create session store
	sessionStore := memory.NewSessionStore()

	// Create use cases
	// Get default model from provider config
	defaultModel := ""
	for _, p := range cfg.AI.Providers {
		if p.Name == cfg.AI.DefaultProvider {
			defaultModel = p.Model
			break
		}
	}
	chatWithAI := usecase.NewChatWithAIUseCase(defaultProvider, sessionStore, logger, translator, defaultModel)
	messageHandler := messaging.NewAIHandlerAdapter(chatWithAI, cfg.AI.SystemPrompt)

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket handler
	wsHandler := websocket.NewHandler(
		messageHandler,
		zerologLogger,
		websocket.Config{
			Host:        cfg.Server.Host,
			Port:        cfg.Server.WSPort,
			AuthEnabled: cfg.WebSocket.Auth.Enabled,
			JWTSecret:   cfg.WebSocket.Auth.JWTSecret,
		},
	)
	mux.HandleFunc("/ws", wsHandler.HandleHTTP)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// POST /api/message — entry point for the messaging runtime (messagingd).
	// It accepts an InboundMessage as JSON, processes it through the AI handler,
	// and returns an OutboundMessage as JSON.
	mux.HandleFunc("/api/message", makeMessageAPIHandler(messageHandler, cfg.Gateway.APIKey))

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("server error: %v", err))
		}
	}()

	logger.Info("ready, waiting for requests...")

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("shutdown error: %v", err))
	}
	logger.Info("goodbye")
}

// makeMessageAPIHandler returns an http.HandlerFunc that receives an InboundMessage
// as JSON, processes it with the given handler, and writes back an OutboundMessage
// as JSON. When apiKey is non-empty, the request must carry a matching
// "Authorization: Bearer <apiKey>" header.
func makeMessageAPIHandler(handler inbound.MessageHandler, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Optional API-key authentication between runtimes.
		if apiKey != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + apiKey
			if auth != expected {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var msg inbound.InboundMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		out, err := handler.Handle(r.Context(), &msg)
		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

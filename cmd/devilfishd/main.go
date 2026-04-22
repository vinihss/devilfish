package main

import (
	"context"
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
	"devilfish/internal/adapters/messaging/discord"
	"devilfish/internal/adapters/messaging/slack"
	"devilfish/internal/adapters/messaging/telegram"
	"devilfish/internal/adapters/storage/memory"
	"devilfish/internal/adapters/websocket"
	"devilfish/internal/application/usecase"
	"devilfish/internal/infra/config"
	"devilfish/internal/infra/i18n"
	"devilfish/internal/infra/logging"
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
		cfg = config.DefaultConfig
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
	messageHandler := messaging.NewAIHandlerAdapter(chatWithAI)

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket handler
	wsHandler := websocket.NewHandler(
		messageHandler,
		zerologLogger,
		websocket.Config{
			Host:        cfg.Server.Host,
			Port:        cfg.Server.WSPort,
			AuthEnabled: cfg.Websocket.Auth.Enabled,
			JWTSecret:   cfg.Websocket.Auth.JWTSecret,
		},
	)
	mux.HandleFunc("/ws", wsHandler.HandleHTTP)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Messaging adapters
	if cfg.Messaging.Telegram.Enabled {
		telegramAdapter := telegram.NewAdapter(cfg.Messaging.Telegram, messageHandler, zerologLogger)
		mux.HandleFunc("/webhooks/telegram", telegramAdapter.HandleWebhook)

		// Start polling - runs until program exits
		go func() {
			telegramCtx := context.Background()
			if err := telegramAdapter.Start(telegramCtx); err != nil {
				logger.Error(fmt.Sprintf("telegram polling stopped: %v", err))
			}
		}()
		logger.Info("Telegram polling started")
	}
	if cfg.Messaging.Discord.Enabled {
		discordAdapter := discord.NewAdapter(cfg.Messaging.Discord, messageHandler, zerologLogger)
		mux.HandleFunc("/webhooks/discord", discordAdapter.HandleWebhook)
		logger.Info("Discord adapter ready")
	}
	if cfg.Messaging.Slack.Enabled {
		slackAdapter := slack.NewAdapter(cfg.Messaging.Slack, messageHandler, zerologLogger)
		mux.HandleFunc("/webhooks/slack", slackAdapter.HandleWebhook)
		logger.Info("Slack adapter ready")
	}

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

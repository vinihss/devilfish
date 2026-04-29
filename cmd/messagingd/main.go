// messagingd is the messaging runtime for DevilFish.
//
// It runs the platform adapters (Telegram, Discord, Slack) and forwards every
// incoming message to the core runtime (devilfishd) via HTTP. The core runtime
// handles AI processing and returns the response, which messagingd then sends
// back to the originating platform.
//
// This separation means the messaging adapters have no direct dependency on AI
// providers, session storage, or MCP servers — they only depend on the
// inbound.MessageHandler interface, which is fulfilled here by the GatewayClient.
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

	"devilfish/internal/adapters/gatewayclient"
	"devilfish/internal/adapters/messaging/discord"
	"devilfish/internal/adapters/messaging/slack"
	"devilfish/internal/adapters/messaging/telegram"
	"devilfish/internal/infra/config"
	"devilfish/internal/infra/logging"
)

func main() {
	// Initialize logger
	zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger := logging.NewLoggerAdapter(&zerologLogger)
	logger.Info("starting DevilFish messaging runtime (messagingd)")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Warn(fmt.Sprintf("could not load config file, using defaults: %v", err))
		cfg = config.DefaultConfig()
	}

	if cfg.Gateway.URL == "" {
		logger.Error("gateway.url must be set in config")
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("forwarding messages to core gateway at %s", cfg.Gateway.URL))

	// Create the gateway client — the single MessageHandler used by all adapters.
	gwTimeout := time.Duration(cfg.Gateway.Timeout) * time.Second
	if gwTimeout <= 0 {
		gwTimeout = 30 * time.Second
	}
	gwClient := gatewayclient.New(gatewayclient.Config{
		URL:     cfg.Gateway.URL,
		APIKey:  cfg.Gateway.APIKey,
		Timeout: gwTimeout,
	})

	// HTTP mux for platform webhooks
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Context cancelled on shutdown — lets Telegram polling stop cleanly.
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	// Register adapters that are enabled
	if cfg.Messaging.Telegram.Enabled {
		telegramAdapter := telegram.NewAdapter(cfg.Messaging.Telegram, gwClient, zerologLogger)
		mux.HandleFunc("/webhooks/telegram", telegramAdapter.HandleWebhook)

		go func() {
			if err := telegramAdapter.Start(runCtx); err != nil {
				logger.Error(fmt.Sprintf("telegram adapter stopped: %v", err))
			}
		}()
		logger.Info("Telegram adapter started")
	}

	if cfg.Messaging.Discord.Enabled {
		discordAdapter := discord.NewAdapter(cfg.Messaging.Discord, gwClient, zerologLogger)
		mux.HandleFunc("/webhooks/discord", discordAdapter.HandleWebhook)
		logger.Info("Discord adapter ready")
	}

	if cfg.Messaging.Slack.Enabled {
		slackAdapter := slack.NewAdapter(cfg.Messaging.Slack, gwClient, zerologLogger)
		mux.HandleFunc("/webhooks/slack", slackAdapter.HandleWebhook)
		logger.Info("Slack adapter ready")
	}

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Messaging.Server.Host, cfg.Messaging.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Messaging.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Messaging.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logger.Info(fmt.Sprintf("messaging server listening on %s", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("messaging server error: %v", err))
		}
	}()

	logger.Info("messaging runtime ready")

	// Graceful shutdown on signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down messaging runtime...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("shutdown error: %v", err))
	}
	logger.Info("messaging runtime stopped")
}

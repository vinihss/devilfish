package config

// Config represents the main configuration structure for DevilFish.
// It includes all sections: Server, AI, Messaging, i18n, Logging, and Security.
type Config struct {
	// Server defines the HTTP/WebSocket server configuration.
	Server ServerConfig `yaml:"server"`

	// AI defines the AI provider configuration.
	AI AIConfig `yaml:"ai"`

	// Messaging defines the messaging channel configurations.
	Messaging MessagingConfig `yaml:"messaging"`

	// I18n defines the internationalization configuration.
	I18n I18nConfig `yaml:"i18n"`

	// Logging defines the logging configuration.
	Logging LoggingConfig `yaml:"logging"`

	// Security defines the security configuration.
	Security SecurityConfig `yaml:"security"`

	// Websocket defines the WebSocket configuration.
	Websocket WebsocketConfig `yaml:"websocket"`
}

// DefaultConfig is the default configuration for DevilFish.
var DefaultConfig = &Config{
	Server: ServerConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		WSPort:       8081,
		ReadTimeout:  30,
		WriteTimeout: 30,
	},
	AI: AIConfig{
		Providers:       []ProviderConfig{},
		DefaultProvider: "",
		Timeout:         30,
		MaxRetries:      3,
	},
	Messaging: MessagingConfig{
		Telegram: TelegramConfig{
			Enabled:   false,
			BotToken:  "",
			AllowList: []string{},
		},
		Discord: DiscordConfig{
			Enabled:   false,
			BotToken:  "",
			AllowList: []string{},
		},
		Slack: SlackConfig{
			Enabled:   false,
			BotToken:  "",
			AppToken:  "",
			AllowList: []string{},
		},
	},
	I18n: I18nConfig{
		DefaultLocale:    "en",
		SupportedLocales: []string{"en"},
		FallbackLocale:   "en",
	},
	Logging: LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	},
	Security: SecurityConfig{
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			Burst:             10,
		},
	},
}

// ServerConfig holds the HTTP server configuration.
type ServerConfig struct {
	// Host is the server bind address.
	Host string `yaml:"host"`

	// Port is the HTTP server port.
	Port int `yaml:"port"`

	// WSPort is the WebSocket server port.
	WSPort int `yaml:"ws_port"`

	// ReadTimeout is the maximum duration for reading the full request.
	ReadTimeout int `yaml:"read_timeout"`

	// WriteTimeout is the maximum duration for writing the response.
	WriteTimeout int `yaml:"write_timeout"`
}

// AIConfig holds the AI provider configuration.
type AIConfig struct {
	// Providers is the list of AI provider configurations.
	Providers []ProviderConfig `yaml:"providers"`

	// DefaultProvider is the default AI provider name.
	DefaultProvider string `yaml:"default_provider"`

	// Timeout is the AI request timeout in seconds.
	Timeout int `yaml:"timeout"`

	// MaxRetries is the maximum number of retries for failed requests.
	MaxRetries int `yaml:"max_retries"`
}

// ProviderConfig holds the configuration for a specific AI provider.
type ProviderConfig struct {
	// Name is the provider name (e.g., "openai", "groq", "gemini", "ollama").
	Name string `yaml:"name"`

	// Enabled indicates whether the provider is enabled.
	Enabled bool `yaml:"enabled"`

	// APIKey is the API key for the provider (use ${PROVIDER_API_KEY} for env vars).
	APIKey string `yaml:"api_key"`

	// BaseURL is the base URL for the provider API (optional).
	BaseURL string `yaml:"base_url"`

	// Model is the model name to use (optional).
	Model string `yaml:"model"`
}

// MessagingConfig holds the messaging channel configurations.
type MessagingConfig struct {
	// Telegram is the Telegram bot configuration.
	Telegram TelegramConfig `yaml:"telegram"`

	// Discord is the Discord bot configuration.
	Discord DiscordConfig `yaml:"discord"`

	// Slack is the Slack bot configuration.
	Slack SlackConfig `yaml:"slack"`
}

// TelegramConfig holds the Telegram bot configuration.
type TelegramConfig struct {
	// Enabled indicates whether the Telegram bot is enabled.
	Enabled bool `yaml:"enabled"`

	// BotToken is the Telegram bot token (use ${TELEGRAM_BOT_TOKEN} for env vars).
	BotToken string `yaml:"bot_token"`

	// AllowList is the list of allowed user IDs (empty = allow all).
	AllowList []string `yaml:"allow_list"`
}

// DiscordConfig holds the Discord bot configuration.
type DiscordConfig struct {
	// Enabled indicates whether the Discord bot is enabled.
	Enabled bool `yaml:"enabled"`

	// BotToken is the Discord bot token (use ${DISCORD_BOT_TOKEN} for env vars).
	BotToken string `yaml:"bot_token"`

	// AllowList is the list of allowed guild/channel IDs (empty = allow all).
	AllowList []string `yaml:"allow_list"`
}

// SlackConfig holds the Slack bot configuration.
type SlackConfig struct {
	// Enabled indicates whether the Slack bot is enabled.
	Enabled bool `yaml:"enabled"`

	// BotToken is the Slack bot token (use ${SLACK_BOT_TOKEN} for env vars).
	BotToken string `yaml:"bot_token"`

	// AppToken is the Slack app token for WebSocket (use ${SLACK_APP_TOKEN} for env vars).
	AppToken string `yaml:"app_token"`

	// AllowList is the list of allowed channel IDs (empty = allow all).
	AllowList []string `yaml:"allow_list"`
}

// I18nConfig holds the internationalization configuration.
type I18nConfig struct {
	// DefaultLocale is the default locale to use.
	DefaultLocale string `yaml:"default_locale"`

	// SupportedLocales is the list of supported locales.
	SupportedLocales []string `yaml:"supported_locales"`

	// FallbackLocale is the fallback locale when translation is missing.
	FallbackLocale string `yaml:"fallback_locale"`
}

// LoggingConfig holds the logging configuration.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error).
	Level string `yaml:"level"`

	// Format is the log format (json, text).
	Format string `yaml:"format"`

	// Output is the log output (stdout, stderr, file).
	Output string `yaml:"output"`
}

// SecurityConfig holds the security configuration.
type SecurityConfig struct {
	// RateLimit is the rate limiting configuration.
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig holds the rate limiting configuration.
type RateLimitConfig struct {
	// Enabled indicates whether rate limiting is enabled.
	Enabled bool `yaml:"enabled"`

	// RequestsPerMinute is the maximum number of requests per minute.
	RequestsPerMinute int `yaml:"requests_per_minute"`

	// Burst is the maximum burst size.
	Burst int `yaml:"burst"`
}

// WebsocketConfig holds the WebSocket configuration.
type WebsocketConfig struct {
	// Server is the WebSocket server configuration.
	Server WebsocketServerConfig `yaml:"server"`

	// Auth is the WebSocket authentication configuration.
	Auth WebsocketAuthConfig `yaml:"auth"`
}

// WebsocketServerConfig holds the WebSocket server configuration.
type WebsocketServerConfig struct {
	// Host is the server bind address.
	Host string `yaml:"host"`

	// Port is the WebSocket server port.
	Port int `yaml:"port"`
}

// WebsocketAuthConfig holds the WebSocket authentication configuration.
type WebsocketAuthConfig struct {
	// Enabled indicates whether authentication is enabled.
	Enabled bool `yaml:"enabled"`

	// JWTSecret is the secret for JWT signing.
	JWTSecret string `yaml:"jwt_secret"`
}

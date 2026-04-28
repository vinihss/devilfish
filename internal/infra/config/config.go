package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	AI         AIConfig        `yaml:"ai"`
	Messaging  MessagingConfig `yaml:"messaging"`
	MCP        MCPConfig       `yaml:"mcp"`
	WebSocket  WebSocketConfig `yaml:"websocket"`
	Logging    LoggingConfig   `yaml:"logging"`
	Security   SecurityConfig  `yaml:"security"`
	I18n       I18nConfig     `yaml:"i18n"`
}

// ServerConfig holds server settings
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	WSPort       int    `yaml:"ws_port"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// AIConfig holds AI provider settings
type AIConfig struct {
	Providers       []ProviderConfig `yaml:"providers"`
	DefaultProvider string           `yaml:"default_provider"`
	// SystemPrompt is the system-level instruction sent to the AI on every request,
	// defining the agent's persona, language, and behavior.
	SystemPrompt string `yaml:"system_prompt"`
	Timeout      int    `yaml:"timeout"`
	MaxRetries   int    `yaml:"max_retries"`
}

// ProviderConfig holds individual provider settings
type ProviderConfig struct {
	Name     string `yaml:"name"`
	Enabled  bool   `yaml:"enabled"`
	APIKey   string `yaml:"api_key"`
	BaseURL  string `yaml:"base_url"`
	Model    string `yaml:"model"`
}

// MessagingConfig holds messaging channel settings
type MessagingConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"`
	Slack    SlackConfig   `yaml:"slack"`
}

// TelegramConfig holds Telegram settings
type TelegramConfig struct {
	Enabled   bool     `yaml:"enabled"`
	BotToken  string  `yaml:"bot_token"`
	AllowList []string `yaml:"allow_list"`
}

// DiscordConfig holds Discord settings
type DiscordConfig struct {
	Enabled   bool     `yaml:"enabled"`
	BotToken  string   `yaml:"bot_token"`
	AllowList []string `yaml:"allow_list"`
}

// SlackConfig holds Slack settings
type SlackConfig struct {
	Enabled   bool     `yaml:"enabled"`
	BotToken  string   `yaml:"bot_token"`
	AppToken string   `yaml:"app_token"`
	AllowList []string `yaml:"allow_list"`
}

// MCPConfig holds MCP server settings
type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"`
}

// MCPServerConfig holds individual MCP server settings
type MCPServerConfig struct {
	Name       string   `yaml:"name"`
	Enabled   bool     `yaml:"enabled"`
	Transport string   `yaml:"transport"`
	URL       string   `yaml:"url"`
	Command   string   `yaml:"command"`
	Args      []string `yaml:"args"`
	AuthToken string   `yaml:"auth_token"`
	Timeout   int      `yaml:"timeout"`
	MaxRetries int    `yaml:"max_retries"`
}

// WebSocketConfig holds WebSocket settings
type WebSocketConfig struct {
	Server ServerSubConfig `yaml:"server"`
	Auth   AuthSubConfig   `yaml:"auth"`
}

// ServerSubConfig holds WebSocket server settings
type ServerSubConfig struct {
	Host string `yaml:"host"`
	Port int   `yaml:"port"`
}

// AuthSubConfig holds authentication settings
type AuthSubConfig struct {
	Enabled    bool   `yaml:"enabled"`
	JWTSecret string `yaml:"jwt_secret"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	Enabled          bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst            int  `yaml:"burst"`
}

// I18nConfig holds internationalization settings
type I18nConfig struct {
	DefaultLocale    string   `yaml:"default_locale"`
	SupportedLocales []string `yaml:"supported_locales"`
	FallbackLocale  string   `yaml:"fallback_locale"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8082,
			WSPort:       8083,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		AI: AIConfig{
			Providers: []ProviderConfig{
				{Name: "openai", Enabled: false, BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
				{Name: "groq", Enabled: false, BaseURL: "https://api.groq.com/openai/v1", Model: "llama-3.1-8b-instant"},
				{Name: "gemini", Enabled: false, BaseURL: "https://generativelanguage.googleapis.com/v1", Model: "gemini-2.0-flash"},
				{Name: "ollama", Enabled: false, BaseURL: "http://localhost:11434", Model: "llama3.2"},
			},
			DefaultProvider: "groq",
			SystemPrompt:    "Você é um assistente de conversação em português brasileiro. Responda sempre em pt-BR de forma clara, amigável e natural.",
			Timeout:         30,
			MaxRetries:      3,
		},
		Messaging: MessagingConfig{
			Telegram: TelegramConfig{Enabled: false, AllowList: []string{}},
			Discord:  DiscordConfig{Enabled: false, AllowList: []string{}},
			Slack:   SlackConfig{Enabled: false, AllowList: []string{}},
		},
		MCP: MCPConfig{
			Servers: []MCPServerConfig{},
		},
		WebSocket: WebSocketConfig{
			Server: ServerSubConfig{Host: "0.0.0.0", Port: 8081},
			Auth:   AuthSubConfig{Enabled: false, JWTSecret: ""},
		},
		Logging: LoggingConfig{Level: "info", Format: "json", Output: "stdout"},
		Security: SecurityConfig{
			RateLimit: RateLimitConfig{Enabled: true, RequestsPerMinute: 60, Burst: 10},
		},
		I18n: I18nConfig{
			DefaultLocale:    "en",
			SupportedLocales: []string{"en", "pt", "es"},
			FallbackLocale:  "en",
		},
	}
}

// Save writes configuration to file
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
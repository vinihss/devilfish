package cli

import (
	"fmt"
	"io"
	"strings"

	"devilfish/internal/infra/config"
)

// PrintConfig prints the configuration to the given writer
func PrintConfig(cfg *config.Config, w io.Writer) {
	fmt.Fprintln(w, "# DevilFish Configuration")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "server:\n")
	fmt.Fprintf(w, "  host: %s\n", cfg.Server.Host)
	fmt.Fprintf(w, "  port: %d\n", cfg.Server.Port)
	fmt.Fprintf(w, "  ws_port: %d\n", cfg.Server.WSPort)
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "ai:\n")
	fmt.Fprintf(w, "  default_provider: %s\n", cfg.AI.DefaultProvider)
	fmt.Fprintf(w, "  timeout: %d\n", cfg.AI.Timeout)
	fmt.Fprintf(w, "  providers:\n")
	for _, p := range cfg.AI.Providers {
		fmt.Fprintf(w, "    - name: %s\n", p.Name)
		fmt.Fprintf(w, "      enabled: %v\n", p.Enabled)
		if p.APIKey != "" {
			fmt.Fprintf(w, "      api_key: %s\n", MaskSecret(p.APIKey))
		}
		if p.BaseURL != "" {
			fmt.Fprintf(w, "      base_url: %s\n", p.BaseURL)
		}
		if p.Model != "" {
			fmt.Fprintf(w, "      model: %s\n", p.Model)
		}
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "messaging:\n")
	if cfg.Messaging.Telegram.Enabled {
		fmt.Fprintf(w, "  telegram:\n")
		fmt.Fprintf(w, "    enabled: true\n")
		if cfg.Messaging.Telegram.BotToken != "" {
			fmt.Fprintf(w, "    bot_token: %s\n", MaskSecret(cfg.Messaging.Telegram.BotToken))
		}
	}
	if cfg.Messaging.Discord.Enabled {
		fmt.Fprintf(w, "  discord:\n")
		fmt.Fprintf(w, "    enabled: true\n")
		if cfg.Messaging.Discord.BotToken != "" {
			fmt.Fprintf(w, "    bot_token: %s\n", MaskSecret(cfg.Messaging.Discord.BotToken))
		}
	}
	if cfg.Messaging.Slack.Enabled {
		fmt.Fprintf(w, "  slack:\n")
		fmt.Fprintf(w, "    enabled: true\n")
		if cfg.Messaging.Slack.BotToken != "" {
			fmt.Fprintf(w, "    bot_token: %s\n", MaskSecret(cfg.Messaging.Slack.BotToken))
		}
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "logging:\n")
	fmt.Fprintf(w, "  level: %s\n", cfg.Logging.Level)
	fmt.Fprintf(w, "  format: %s\n", cfg.Logging.Format)
	fmt.Fprintf(w, "  output: %s\n", cfg.Logging.Output)
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "security:\n")
	fmt.Fprintf(w, "  rate_limit:\n")
	fmt.Fprintf(w, "    enabled: %v\n", cfg.Security.RateLimit.Enabled)
	fmt.Fprintf(w, "    requests_per_minute: %d\n", cfg.Security.RateLimit.RequestsPerMinute)
	fmt.Fprintf(w, "    burst: %d\n", cfg.Security.RateLimit.Burst)
}

// GetConfigValue returns a config value by key
func GetConfigValue(cfg *config.Config, key string) string {
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "server":
		if len(parts) < 2 {
			return ""
		}
		switch parts[1] {
		case "host":
			return cfg.Server.Host
		case "port":
			return fmt.Sprintf("%d", cfg.Server.Port)
		case "ws_port":
			return fmt.Sprintf("%d", cfg.Server.WSPort)
		case "read_timeout":
			return fmt.Sprintf("%d", cfg.Server.ReadTimeout)
		case "write_timeout":
			return fmt.Sprintf("%d", cfg.Server.WriteTimeout)
		}
	case "ai":
		if len(parts) < 2 {
			return ""
		}
		switch parts[1] {
		case "default_provider":
			return cfg.AI.DefaultProvider
		case "timeout":
			return fmt.Sprintf("%d", cfg.AI.Timeout)
		case "max_retries":
			return fmt.Sprintf("%d", cfg.AI.MaxRetries)
		}
	case "logging":
		if len(parts) < 2 {
			return ""
		}
		switch parts[1] {
		case "level":
			return cfg.Logging.Level
		case "format":
			return cfg.Logging.Format
		case "output":
			return cfg.Logging.Output
		}
	case "security":
		if len(parts) < 2 {
			return ""
		}
		if parts[1] == "rate_limit" && len(parts) >= 3 {
			switch parts[2] {
			case "enabled":
				return fmt.Sprintf("%v", cfg.Security.RateLimit.Enabled)
			case "requests_per_minute":
				return fmt.Sprintf("%d", cfg.Security.RateLimit.RequestsPerMinute)
			case "burst":
				return fmt.Sprintf("%d", cfg.Security.RateLimit.Burst)
			}
		}
	case "i18n":
		if len(parts) < 2 {
			return ""
		}
		switch parts[1] {
		case "default_locale":
			return cfg.I18n.DefaultLocale
		case "fallback_locale":
			return cfg.I18n.FallbackLocale
		}
	}
	return ""
}

// SetConfigValue sets a config value by key
func SetConfigValue(cfg *config.Config, key, value string) error {
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid key")
	}

	switch parts[0] {
	case "server":
		if len(parts) < 2 {
			return fmt.Errorf("invalid server key")
		}
		switch parts[1] {
		case "host":
			cfg.Server.Host = value
		case "port":
			var port int
			fmt.Sscanf(value, "%d", &port)
			cfg.Server.Port = port
		case "ws_port":
			var port int
			fmt.Sscanf(value, "%d", &port)
			cfg.Server.WSPort = port
		case "read_timeout":
			var to int
			fmt.Sscanf(value, "%d", &to)
			cfg.Server.ReadTimeout = to
		case "write_timeout":
			var to int
			fmt.Sscanf(value, "%d", &to)
			cfg.Server.WriteTimeout = to
		default:
			return fmt.Errorf("unknown server key: %s", parts[1])
		}
	case "ai":
		if len(parts) < 2 {
			return fmt.Errorf("invalid ai key")
		}
		switch parts[1] {
		case "default_provider":
			cfg.AI.DefaultProvider = value
		case "timeout":
			var to int
			fmt.Sscanf(value, "%d", &to)
			cfg.AI.Timeout = to
		case "max_retries":
			var r int
			fmt.Sscanf(value, "%d", &r)
			cfg.AI.MaxRetries = r
		default:
			return fmt.Errorf("unknown ai key: %s", parts[1])
		}
	case "logging":
		if len(parts) < 2 {
			return fmt.Errorf("invalid logging key")
		}
		switch parts[1] {
		case "level":
			cfg.Logging.Level = value
		case "format":
			cfg.Logging.Format = value
		case "output":
			cfg.Logging.Output = value
		default:
			return fmt.Errorf("unknown logging key: %s", parts[1])
		}
	case "security":
		if len(parts) < 2 || parts[1] != "rate_limit" {
			return fmt.Errorf("invalid security key")
		}
		if len(parts) < 3 {
			return fmt.Errorf("invalid rate_limit key")
		}
		switch parts[2] {
		case "enabled":
			cfg.Security.RateLimit.Enabled = value == "true"
		case "requests_per_minute":
			var r int
			fmt.Sscanf(value, "%d", &r)
			cfg.Security.RateLimit.RequestsPerMinute = r
		case "burst":
			var b int
			fmt.Sscanf(value, "%d", &b)
			cfg.Security.RateLimit.Burst = b
		default:
			return fmt.Errorf("unknown rate_limit key: %s", parts[2])
		}
	case "i18n":
		if len(parts) < 2 {
			return fmt.Errorf("invalid i18n key")
		}
		switch parts[1] {
		case "default_locale":
			cfg.I18n.DefaultLocale = value
		case "fallback_locale":
			cfg.I18n.FallbackLocale = value
		default:
			return fmt.Errorf("unknown i18n key: %s", parts[1])
		}
	default:
		return fmt.Errorf("unknown key: %s", parts[0])
	}
	return nil
}

// MaskSecret masks a secret value for display
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:4] + "****"
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Printf("✗ %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("ℹ %s\n", message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Printf("⚠ %s\n", message)
}
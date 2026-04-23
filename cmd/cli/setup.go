package main

import (
	"fmt"
	"os"

	"devilfish/internal/infra/config"
	"devilfish/internal/infra/cli"

	"github.com/spf13/cobra"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  `Run an interactive wizard to configure AI providers, messaging channels, and MCP servers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupWizard()
	},
}

func runSetupWizard() error {
	// Create prompts instance
	prompts := cli.NewPrompts()
	validator := cli.NewConfigValidator()

	// Banner
	printBanner()

	// Start wizard
	fmt.Println("\nWelcome to the DevilFish Setup Wizard!")
	fmt.Println("This wizard will help you configure your AI providers, messaging channels, and MCP servers.")
	fmt.Println()

	// Check if config file exists
	configPath := "config.yaml"
	existingConfig := config.DefaultConfig()
	
	if _, err := os.Stat(configPath); err == nil {
		// Load existing config
		var err error
		existingConfig, err = config.LoadFromFile(configPath)
		if err != nil {
			cli.PrintWarning(fmt.Sprintf("Could not load existing config: %v. Using defaults.", err))
			existingConfig = config.DefaultConfig()
		}
	}

	// Ask to modify existing or create new
	var modifyExisting bool
	modifyExisting, err := prompts.AskYesNo("Do you want to modify the existing configuration?", true)
	if err != nil {
		return err
	}

	cfg := existingConfig
	if !modifyExisting {
		cfg = config.DefaultConfig()
	}

	// Step 1: Server Configuration
	fmt.Println("\n=== Step 1: Server Configuration ===")
	if err := runServerSetup(prompts, cfg); err != nil {
		return fmt.Errorf("server setup failed: %w", err)
	}

	// Step 2: AI Providers
	fmt.Println("\n=== Step 2: AI Providers ===")
	if err := runAIProviderSetup(prompts, validator, cfg); err != nil {
		return fmt.Errorf("AI provider setup failed: %w", err)
	}

	// Step 3: Messaging Channels
	fmt.Println("\n=== Step 3: Messaging Channels ===")
	if err := runMessagingSetup(prompts, validator, cfg); err != nil {
		return fmt.Errorf("messaging setup failed: %w", err)
	}

	// Step 4: MCP Servers (optional)
	fmt.Println("\n=== Step 4: MCP Servers (optional) ===")
	var addMCPServers bool
	addMCPServers, err = prompts.AskYesNo("Do you want to configure MCP servers?", false)
	if err != nil {
		return err
	}
	if addMCPServers {
		if err := runMCPSetup(prompts, validator, cfg); err != nil {
			return fmt.Errorf("MCP setup failed: %w", err)
		}
	}

	// Step 5: Security & Logging
	fmt.Println("\n=== Step 5: Security & Logging ===")
	if err := runSecurityLoggingSetup(prompts, cfg); err != nil {
		return fmt.Errorf("security/logging setup failed: %w", err)
	}

	// Save configuration
	fmt.Println("\n=== Saving Configuration ===")
	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess(fmt.Sprintf("Configuration saved to %s", configPath))

	// Validate
	fmt.Println("\n=== Validating Configuration ===")
	if err := cfg.Validate(); err != nil {
		cli.PrintWarning(fmt.Sprintf("Validation warning: %v", err))
	} else {
		cli.PrintSuccess("Configuration is valid")
	}

	// Summary
	fmt.Println("\n=== Setup Complete ===")
	printSummary(cfg)

	return nil
}

func printBanner() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ███████╗ ██████╗ ███╗   ███╗██╗   ██╗███████╗██╗      ║
║   ██╔════╝██╔═══██╗████╗ ████║██║   ██║██╔════╝██║      ║
║   █████╗  ██║   ██║██╔████╔██║██║   ██║███████╗██║      ║
║   ██╔══╝  ██║   ██║██║╚██╔╝██║██║   ██║╚════██║██║      ║
║   ██║     ╚██████╔╝██║ ╚═╝ ██║╚██████╔╝███████╗███████╗║
║   ╚═╝      ╚═════╝ ╚═╝     ╚═╝ ╚═════╝ ╚══════╝╚══════╝║
║                                                               ║
║             Setup Wizard v1.0.0                                ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`)
}

func runServerSetup(p *cli.Prompts, cfg *config.Config) error {
	var err error

	// Host
	cfg.Server.Host, err = p.AskString("Server host", cfg.Server.Host)
	if err != nil {
		return err
	}

	// Port
	var portStr string
	portStr, err = p.AskString("Server HTTP port", fmt.Sprintf("%d", cfg.Server.Port))
	if err != nil {
		return err
	}
	fmt.Sscanf(portStr, "%d", &cfg.Server.Port)

	// WebSocket Port
	var wsPortStr string
	wsPortStr, err = p.AskString("Server WebSocket port", fmt.Sprintf("%d", cfg.Server.WSPort))
	if err != nil {
		return err
	}
	fmt.Sscanf(wsPortStr, "%d", &cfg.Server.WSPort)

	return nil
}

func runAIProviderSetup(p *cli.Prompts, v *cli.ConfigValidator, cfg *config.Config) error {
	availableProviders := []string{"openai", "groq", "gemini", "ollama"}

	// Ask which providers to enable
	selectedProviders, err := p.AskMultiSelect("Select AI providers to enable:", availableProviders)
	if err != nil {
		return err
	}

	// Reset providers and add selected ones
	cfg.AI.Providers = nil

	for _, providerName := range selectedProviders {
		provider := config.ProviderConfig{
			Name:    providerName,
			Enabled: true,
		}

		// Set defaults based on provider
		switch providerName {
		case "openai":
			provider.BaseURL = "https://api.openai.com/v1"
			provider.Model = "gpt-4o"
		case "groq":
			provider.BaseURL = "https://api.groq.com/openai/v1"
			provider.Model = "llama-3.1-8b-instant"
		case "gemini":
			provider.BaseURL = "https://generativelanguage.googleapis.com/v1"
			provider.Model = "gemini-2.0-flash"
		case "ollama":
			provider.BaseURL = "http://localhost:11434"
			provider.Model = "llama3.2"
		}

		// Ask for API key (optional for ollama)
		if providerName != "ollama" {
			var apiKey string
			apiKey, err = p.AskPassword(fmt.Sprintf("API key for %s:", providerName))
			if err != nil {
				return err
			}

			// Validate
			if err := v.ValidateProviderAPIKey(providerName, apiKey); err != nil {
				cli.PrintWarning(fmt.Sprintf("API key validation: %v", err))
			}
			provider.APIKey = apiKey
		}

		// Ask for model
		var model string
		model, err = p.AskString("Model", provider.Model)
		if err != nil {
			return err
		}
		provider.Model = model

		cfg.AI.Providers = append(cfg.AI.Providers, provider)
	}

	// Default provider
	defaultIdx := 0
	for i, p := range cfg.AI.Providers {
		if p.Name == cfg.AI.DefaultProvider {
			defaultIdx = i
			break
		}
	}

	providerNames := make([]string, len(cfg.AI.Providers))
	for i, p := range cfg.AI.Providers {
		providerNames[i] = p.Name
	}

	defaultProvider, err := p.AskSelect("Default provider:", providerNames, defaultIdx)
	if err != nil {
		return err
	}
	cfg.AI.DefaultProvider = defaultProvider

	return nil
}

func runMessagingSetup(p *cli.Prompts, v *cli.ConfigValidator, cfg *config.Config) error {
	availableChannels := []string{"telegram", "discord", "slack"}

	// Ask which channels to enable
	selectedChannels, err := p.AskMultiSelect("Select messaging channels to enable:", availableChannels)
	if err != nil {
		return err
	}

	// Reset channels
	cfg.Messaging.Telegram.Enabled = false
	cfg.Messaging.Discord.Enabled = false
	cfg.Messaging.Slack.Enabled = false

	for _, channelName := range selectedChannels {
		switch channelName {
		case "telegram":
			cfg.Messaging.Telegram.Enabled = true
			botToken, err := p.AskPassword("Telegram bot token:")
			if err != nil {
				return err
			}
			if err := v.ValidateBotToken(botToken); err != nil {
				cli.PrintWarning(fmt.Sprintf("Token validation: %v", err))
			}
			cfg.Messaging.Telegram.BotToken = botToken

		case "discord":
			cfg.Messaging.Discord.Enabled = true
			botToken, err := p.AskPassword("Discord bot token:")
			if err != nil {
				return err
			}
			if err := v.ValidateBotToken(botToken); err != nil {
				cli.PrintWarning(fmt.Sprintf("Token validation: %v", err))
			}
			cfg.Messaging.Discord.BotToken = botToken

		case "slack":
			cfg.Messaging.Slack.Enabled = true
			botToken, err := p.AskPassword("Slack bot token:")
			if err != nil {
				return err
			}
			cfg.Messaging.Slack.BotToken = botToken

			appToken, err := p.AskPassword("Slack app token (xoxb-):")
			if err != nil {
				return err
			}
			cfg.Messaging.Slack.AppToken = appToken
		}
	}

	return nil
}

func runMCPSetup(p *cli.Prompts, v *cli.ConfigValidator, cfg *config.Config) error {
	availableServers := []string{"filesystem", "github", "memory", "custom"}

	// Ask which servers to add
	selectedServers, err := p.AskMultiSelect("Select MCP servers to add:", availableServers)
	if err != nil {
		return err
	}

	// Reset servers
	cfg.MCP.Servers = nil

	for _, serverName := range selectedServers {
		mcpServer := config.MCPServerConfig{
			Name:    serverName,
			Enabled: true,
		}

		// Ask for transport type
		transport, err := p.AskSelect("Transport type:", []string{"stdio", "http"}, 0)
		if err != nil {
			return err
		}
		mcpServer.Transport = transport

		if transport == "stdio" {
			// Command for stdio
			command, err := p.AskString("Command", serverName)
			if err != nil {
				return err
			}
			mcpServer.Command = command
		} else {
			// URL for HTTP
			url, err := p.AskString("Server URL", "http://localhost:3000")
			if err != nil {
				return err
			}
			if err := v.ValidateURL(url); err != nil {
				cli.PrintWarning(fmt.Sprintf("URL validation: %v", err))
			}
			mcpServer.URL = url
		}

		// Auth token (optional)
		var authToken string
		authToken, err = p.AskPassword("Auth token (optional, press Enter to skip):")
		if err != nil {
			return err
		}
		mcpServer.AuthToken = authToken

		cfg.MCP.Servers = append(cfg.MCP.Servers, mcpServer)
	}

	return nil
}

func runSecurityLoggingSetup(p *cli.Prompts, cfg *config.Config) error {
	var err error

	// Logging level
	levels := []string{"debug", "info", "warn", "error"}
	logLevel, err := p.AskSelect("Logging level:", levels, 1)
	if err != nil {
		return err
	}
	cfg.Logging.Level = logLevel

	// Logging format
	formats := []string{"json", "text"}
	logFormat, err := p.AskSelect("Logging format:", formats, 0)
	if err != nil {
		return err
	}
	cfg.Logging.Format = logFormat

	// Rate limiting
	enableRateLimit, err := p.AskYesNo("Enable rate limiting?", true)
	if err != nil {
		return err
	}
	cfg.Security.RateLimit.Enabled = enableRateLimit

	if enableRateLimit {
		var rpmStr string
		rpmStr, err = p.AskString("Requests per minute", fmt.Sprintf("%d", cfg.Security.RateLimit.RequestsPerMinute))
		if err != nil {
			return err
		}
		fmt.Sscanf(rpmStr, "%d", &cfg.Security.RateLimit.RequestsPerMinute)

		var burstStr string
		burstStr, err = p.AskString("Burst", fmt.Sprintf("%d", cfg.Security.RateLimit.Burst))
		if err != nil {
			return err
		}
		fmt.Sscanf(burstStr, "%d", &cfg.Security.RateLimit.Burst)
	}

	// i18n
	locales := []string{"en", "pt", "es", "zh"}
	defaultLocale, err := p.AskSelect("Default locale:", locales, 0)
	if err != nil {
		return err
	}
	cfg.I18n.DefaultLocale = defaultLocale

	return nil
}

func printSummary(cfg *config.Config) {
	fmt.Println("Configuration Summary:")
	fmt.Println("--------------------------")
	fmt.Printf("Server: %s:%d (WS: %d)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.WSPort)
	fmt.Printf("AI Providers: %d enabled\n", len(cfg.AI.Providers))
	for _, p := range cfg.AI.Providers {
		apiKeyDisplay := "no API key"
		if p.APIKey != "" {
			apiKeyDisplay = cli.MaskSecret(p.APIKey)
		}
		fmt.Printf("  - %s: model=%s, api_key=%s\n", p.Name, p.Model, apiKeyDisplay)
	}
	fmt.Printf("Default Provider: %s\n", cfg.AI.DefaultProvider)

	messagingCount := 0
	if cfg.Messaging.Telegram.Enabled {
		messagingCount++
	}
	if cfg.Messaging.Discord.Enabled {
		messagingCount++
	}
	if cfg.Messaging.Slack.Enabled {
		messagingCount++
	}
	fmt.Printf("Messaging Channels: %d enabled\n", messagingCount)

	fmt.Printf("MCP Servers: %d configured\n", len(cfg.MCP.Servers))
	for _, s := range cfg.MCP.Servers {
		fmt.Printf("  - %s (%s)\n", s.Name, s.Transport)
	}

	fmt.Printf("Logging: %s (%s)\n", cfg.Logging.Level, cfg.Logging.Format)
	fmt.Printf("Rate Limiting: %v\n", cfg.Security.RateLimit.Enabled)
	if cfg.Security.RateLimit.Enabled {
		fmt.Printf("  - %d requests/minute, burst=%d\n", cfg.Security.RateLimit.RequestsPerMinute, cfg.Security.RateLimit.Burst)
	}
}
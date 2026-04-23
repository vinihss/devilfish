package main

import (
	"fmt"
	"os"

	"devilfish/internal/infra/config"
	"devilfish/internal/infra/cli"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg    *config.Config
)

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "devilfish",
	Short: "DevilFish - Messaging harness with AI",
	Long:  `DevilFish is a messaging harness that connects message channels to AI models via WebSocket gateway.`,
}

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Commands to view and modify configuration.`,
}

func loadConfig() (*config.Config, error) {
	if cfgFile != "" {
		return config.LoadFromFile(cfgFile)
	}
	return config.Load()
}

func saveConfig(cfg *config.Config, path string) error {
	return config.Save(path, cfg)
}

// configShowCmd displays all configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		cli.PrintConfig(c, os.Stdout)
		return nil
	},
}

// configGetCmd gets a specific config value
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		val := cli.GetConfigValue(c, args[0])
		if val == "" {
			return fmt.Errorf("key not found: %s", args[0])
		}
		fmt.Println(val)
		return nil
	},
}

// configSetCmd sets a config value
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		if err := cli.SetConfigValue(c, args[0], args[1]); err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}
		
		// Save to file
		path := cfgFile
		if path == "" {
			path = "config.yaml"
		}
		if err := saveConfig(c, path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		fmt.Printf("Set %s = %s\n", args[0], args[1])
		return nil
	},
}

// configValidateCmd validates configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config file: %v\n", err)
			fmt.Println("Using default configuration for validation...")
			c = config.DefaultConfig()
		}
		
		if err := c.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
		
		cli.PrintSuccess("Configuration is valid")
		return nil
	},
}

// configInitCmd initializes a new config file
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			path = "config.yaml"
		}
		
		// Check if file already exists
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists: %s", path)
		}
		
		c := config.DefaultConfig()
		if err := config.Save(path, c); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		cli.PrintSuccess(fmt.Sprintf("Created config file: %s", path))
		return nil
	},
}

func main() {
	// Add flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file path")

	// Add subcommands
	AddSetupCommand(rootCmd)
	rootCmd.AddCommand(configCmd)

	// Config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runSetupWizard is defined in setup.go
func AddSetupCommand(root *cobra.Command) {
	root.AddCommand(setupCmd)
}
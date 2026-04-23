package cli

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// Prompts provides interactive prompt utilities for the setup wizard
type Prompts struct{}

// NewPrompts creates a new Prompts instance
func NewPrompts() *Prompts {
	return &Prompts{}
}

// AskYesNo asks a yes/no question
func (p *Prompts) AskYesNo(question string, defaultVal bool) (bool, error) {
	confirm := &survey.Confirm{
		Message: question,
		Default: defaultVal,
	}
	var result bool
	err := survey.AskOne(confirm, &result)
	return result, err
}

// AskString asks for a string value
func (p *Prompts) AskString(question, defaultVal string) (string, error) {
	input := &survey.Input{
		Message: question,
		Default: defaultVal,
	}
	var result string
	err := survey.AskOne(input, &result)
	return result, err
}

// AskPassword asks for a password (hidden input)
func (p *Prompts) AskPassword(question string) (string, error) {
	input := &survey.Password{
		Message: question,
	}
	var result string
	err := survey.AskOne(input, &result)
	return result, err
}

// AskSelect asks to select one option from a list
func (p *Prompts) AskSelect(question string, options []string, defaultIdx int) (string, error) {
	prompt := &survey.Select{
		Message: question,
		Options: options,
		Default: options[defaultIdx],
	}
	var result string
	err := survey.AskOne(prompt, &result)
	return result, err
}

// AskMultiSelect asks to select multiple options from a list
func (p *Prompts) AskMultiSelect(question string, options []string) ([]string, error) {
	prompt := &survey.MultiSelect{
		Message: question,
		Options: options,
	}
	var result []string
	err := survey.AskOne(prompt, &result)
	return result, err
}

// AskNumber asks for a number
func (p *Prompts) AskNumber(question string, defaultVal float64) (float64, error) {
	input := &survey.Input{
		Message: question,
		Default: fmt.Sprintf("%v", defaultVal),
	}
	var resultStr string
	err := survey.AskOne(input, &resultStr)
	if err != nil {
		return 0, err
	}
	var result float64
	fmt.Sscanf(resultStr, "%f", &result)
	return result, nil
}

// PrintSuccess prints a success message
func (p *Prompts) PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

// PrintError prints an error message
func (p *Prompts) PrintError(message string) {
	fmt.Printf("✗ %s\n", message)
}

// PrintInfo prints an info message
func (p *Prompts) PrintInfo(message string) {
	fmt.Printf("ℹ %s\n", message)
}

// PrintWarning prints a warning message
func (p *Prompts) PrintWarning(message string) {
	fmt.Printf("⚠ %s\n", message)
}

// PromptAIProvider asks for AI provider selection
func (p *Prompts) PromptAIProvider() (string, error) {
	providers := []string{"openai", "groq", "gemini", "ollama"}
	return p.AskSelect("Select AI provider:", providers, 0)
}

// PromptMessagingChannel asks for messaging channel selection
func (p *Prompts) PromptMessagingChannel() (string, error) {
	channels := []string{"telegram", "discord", "slack"}
	return p.AskSelect("Select messaging channel:", channels, 0)
}

// PromptMCPTransport asks for MCP transport type
func (p *Prompts) PromptMCPTransport() (string, error) {
	transports := []string{"stdio", "http"}
	return p.AskSelect("Select MCP transport type:", transports, 0)
}

// PromptMCPServer asks for MCP server type
func (p *Prompts) PromptMCPServer() (string, error) {
	servers := []string{"filesystem", "github", "memory", "custom"}
	return p.AskSelect("Select MCP server type:", servers, 0)
}
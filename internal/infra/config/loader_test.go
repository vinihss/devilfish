package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromFile_WithPlainSystemPrompt_KeepsPromptText(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.AI.SystemPrompt = "Você é um assistente."

	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	loaded, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if loaded.AI.SystemPrompt != cfg.AI.SystemPrompt {
		t.Fatalf("expected plain prompt to stay unchanged, got %q", loaded.AI.SystemPrompt)
	}
}

func TestLoadFromFile_WithFileSystemPrompt_LoadsPromptFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	promptDir := filepath.Join(tmpDir, "prompts", "agents")
	promptPath := filepath.Join(promptDir, "default.txt")
	promptText := "test prompt loaded from file\n"

	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte(promptText), 0o644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AI.SystemPrompt = "file://prompts/agents/default.txt"
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	loaded, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if loaded.AI.SystemPrompt != promptText {
		t.Fatalf("expected prompt from file %q, got %q", promptText, loaded.AI.SystemPrompt)
	}
}

func TestLoadFromFile_WithMissingSystemPromptFile_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.AI.SystemPrompt = "file://prompts/agents/missing.txt"
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}

	if !strings.Contains(err.Error(), "failed to read system prompt file") {
		t.Fatalf("expected missing prompt file error, got: %v", err)
	}
}

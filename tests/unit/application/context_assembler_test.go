package usecase_test

import (
	"strings"
	"testing"
	"time"

	"devilfish/internal/application/usecase"
	"devilfish/internal/ports/outbound"
)

// helper builds a slice of outbound.Message with arbitrary content.
func makeMessages(pairs ...string) []outbound.Message {
	msgs := make([]outbound.Message, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		msgs = append(msgs, outbound.Message{
			Role:    pairs[i],
			Content: pairs[i+1],
			Time:    time.Now(),
		})
	}
	return msgs
}

func TestContextAssembler_Build_WithSystemPromptOnly_ReturnsSystemAndUser(t *testing.T) {
	ca := usecase.NewContextAssembler(usecase.DefaultContextAssemblerConfig())

	input := usecase.AssembleInput{
		SystemPrompt: "You are a helpful assistant.",
		UserInput:    "Hello!",
	}

	msgs := ca.Build(input)

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected first role=system, got %s", msgs[0].Role)
	}
	if msgs[0].Content != "You are a helpful assistant." {
		t.Errorf("unexpected system content: %s", msgs[0].Content)
	}
	if msgs[1].Role != "user" {
		t.Errorf("expected last role=user, got %s", msgs[1].Role)
	}
	if msgs[1].Content != "Hello!" {
		t.Errorf("unexpected user content: %s", msgs[1].Content)
	}
}

func TestContextAssembler_Build_WithRecentMessages_OrderedChronologically(t *testing.T) {
	ca := usecase.NewContextAssembler(usecase.DefaultContextAssemblerConfig())

	recent := makeMessages(
		"user", "first message",
		"assistant", "first response",
		"user", "second message",
	)

	input := usecase.AssembleInput{
		RecentMessages: recent,
		UserInput:      "third message",
	}

	msgs := ca.Build(input)

	// Expect: 3 recent + 1 user = 4 messages (no system prompt)
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "first message" {
		t.Errorf("expected first message first, got: %s", msgs[0].Content)
	}
	if msgs[2].Content != "second message" {
		t.Errorf("expected second message third, got: %s", msgs[2].Content)
	}
	if msgs[3].Role != "user" || msgs[3].Content != "third message" {
		t.Errorf("expected user input last, got role=%s content=%s", msgs[3].Role, msgs[3].Content)
	}
}

func TestContextAssembler_Build_WithRelevantMemory_InjectsMemoryBlock(t *testing.T) {
	ca := usecase.NewContextAssembler(usecase.DefaultContextAssemblerConfig())

	memory := []*outbound.StoredEmbedding{
		{MessageID: "m1", SessionID: "s1", Content: "worker retry strategy discussion"},
		{MessageID: "m2", SessionID: "s1", Content: "queue depth recommendation"},
	}

	input := usecase.AssembleInput{
		SystemPrompt:   "You are an assistant.",
		RelevantMemory: memory,
		UserInput:      "How can I improve my worker system?",
	}

	msgs := ca.Build(input)

	// Expect: system prompt + memory block + user input = 3
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if !strings.Contains(msgs[1].Content, "[RELEVANT MEMORY]") {
		t.Errorf("expected memory block in second message, got: %s", msgs[1].Content)
	}
	if !strings.Contains(msgs[1].Content, "worker retry strategy discussion") {
		t.Errorf("expected first memory entry in block, got: %s", msgs[1].Content)
	}
	if msgs[2].Role != "user" {
		t.Errorf("expected last message role=user, got %s", msgs[2].Role)
	}
}

func TestContextAssembler_Build_WithNoInputs_ReturnsOnlyUserMessage(t *testing.T) {
	ca := usecase.NewContextAssembler(usecase.DefaultContextAssemblerConfig())

	input := usecase.AssembleInput{UserInput: "Hello"}

	msgs := ca.Build(input)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "Hello" {
		t.Errorf("unexpected message: %+v", msgs[0])
	}
}

func TestContextAssembler_Build_TokenBudgetDropsOldMessages(t *testing.T) {
	// Use a very small token budget so most messages are dropped.
	cfg := usecase.ContextAssemblerConfig{MaxTokens: 20}
	ca := usecase.NewContextAssembler(cfg)

	// Each message is roughly 5-10 tokens; with MaxTokens=20 the recent
	// budget (60% = 12 tokens) should only fit the most recent one.
	recent := makeMessages(
		"user", "this is the oldest message in the conversation history",
		"assistant", "this is the oldest assistant reply in the conversation",
		"user", "newest",
	)

	input := usecase.AssembleInput{
		RecentMessages: recent,
		UserInput:      "Hi",
	}

	msgs := ca.Build(input)

	// The newest recent message ("newest") should be included.
	foundNewest := false
	for _, m := range msgs {
		if m.Content == "newest" {
			foundNewest = true
		}
	}
	if !foundNewest {
		t.Error("expected newest recent message to be included")
	}

	// The oldest messages should have been dropped.
	for _, m := range msgs {
		if m.Content == "this is the oldest message in the conversation history" {
			t.Error("oldest message should have been dropped due to token budget")
		}
	}
}

func TestEstimateTokens_ReturnsLengthDividedByFour(t *testing.T) {
	cases := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"abcd", 1},
		{"abcdefgh", 2},
		{"1234567890123456", 4},
	}
	for _, tc := range cases {
		got := usecase.EstimateTokens(tc.text)
		if got != tc.expected {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tc.text, got, tc.expected)
		}
	}
}

func TestContextAssembler_Build_WithZeroMaxTokens_FallsBackToDefault(t *testing.T) {
	// A zero MaxTokens should fall back to the default so no panic occurs.
	ca := usecase.NewContextAssembler(usecase.ContextAssemblerConfig{MaxTokens: 0})

	msgs := ca.Build(usecase.AssembleInput{
		SystemPrompt: "sys",
		UserInput:    "hi",
	})

	if len(msgs) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(msgs))
	}
}

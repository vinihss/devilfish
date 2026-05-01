package usecase

import (
	"devilfish/internal/ports/outbound"
)

// tokenBudget defines the fraction of the total token budget allocated to each
// section of the composed context window.
const (
	systemBudgetFraction   = 0.10 // system instructions
	memoryBudgetFraction   = 0.20 // relevant long-term memory
	recentBudgetFraction   = 0.60 // recent conversation timeline
	responseBudgetFraction = 0.10 // reserved for the model response (not consumed by Build)
)

// ContextAssemblerConfig holds configuration for the ContextAssembler.
type ContextAssemblerConfig struct {
	// MaxTokens is the total token budget for the composed context window.
	// Tokens are estimated using EstimateTokens.
	MaxTokens int
}

// DefaultContextAssemblerConfig returns sensible defaults.
func DefaultContextAssemblerConfig() ContextAssemblerConfig {
	return ContextAssemblerConfig{
		MaxTokens: 4096,
	}
}

// ContextAssembler builds the final context window sent to the LLM.
//
// It composes the context following a structured token budget:
//   - 10% → system instructions
//   - 20% → semantically relevant long-term memory
//   - 60% → recent conversation timeline
//   - 10% → reserved for the model response
type ContextAssembler struct {
	cfg ContextAssemblerConfig
}

// NewContextAssembler creates a new ContextAssembler with the given config.
func NewContextAssembler(cfg ContextAssemblerConfig) *ContextAssembler {
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = DefaultContextAssemblerConfig().MaxTokens
	}
	return &ContextAssembler{cfg: cfg}
}

// AssembleInput holds all inputs required to build a context window.
type AssembleInput struct {
	// SystemPrompt is the static system-level instruction for the model.
	SystemPrompt string

	// RecentMessages is the ordered timeline of the current session (oldest first).
	RecentMessages []outbound.Message

	// RelevantMemory contains semantically retrieved past interactions.
	// These are injected as a structured memory block before the recent timeline.
	RelevantMemory []*outbound.StoredEmbedding

	// UserInput is the current user message being processed.
	UserInput string
}

// Build constructs the ordered list of chat messages that will be sent to the LLM.
//
// The resulting slice follows the format:
//  1. System prompt message (if present)
//  2. Relevant memory block (formatted as a single system message, if present)
//  3. Recent conversation messages (token-aware, oldest messages dropped first)
//  4. Current user input
//
// The remaining 10% of the token budget (responseBudgetFraction) is intentionally
// left unconsumed to provide headroom for the model's response.
func (ca *ContextAssembler) Build(input AssembleInput) []outbound.ChatMessage {
	systemBudget := int(float64(ca.cfg.MaxTokens) * systemBudgetFraction)
	memoryBudget := int(float64(ca.cfg.MaxTokens) * memoryBudgetFraction)
	recentBudget := int(float64(ca.cfg.MaxTokens) * recentBudgetFraction)

	var messages []outbound.ChatMessage

	// 1. System prompt
	if input.SystemPrompt != "" {
		prompt := truncateToTokenBudget(input.SystemPrompt, systemBudget)
		messages = append(messages, outbound.ChatMessage{
			Role:    "system",
			Content: prompt,
		})
	}

	// 2. Relevant memory block
	if len(input.RelevantMemory) > 0 {
		memoryBlock := formatMemoryBlock(input.RelevantMemory, memoryBudget)
		if memoryBlock != "" {
			messages = append(messages, outbound.ChatMessage{
				Role:    "system",
				Content: memoryBlock,
			})
		}
	}

	// 3. Recent conversation — drop oldest messages when over budget
	recent := fitRecentMessages(input.RecentMessages, recentBudget)
	messages = append(messages, recent...)

	// 4. Current user input (always included — it is the reason for the call)
	messages = append(messages, outbound.ChatMessage{
		Role:    "user",
		Content: input.UserInput,
	})

	return messages
}

// EstimateTokens returns a fast approximation of the token count for text.
// Uses the well-known heuristic: 1 token ≈ 4 characters.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// truncateToTokenBudget returns the text truncated so that its estimated
// token count does not exceed budget. When budget is zero or negative the
// original text is returned unchanged.
func truncateToTokenBudget(text string, budget int) string {
	if budget <= 0 {
		return text
	}
	maxChars := budget * 4
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}

// formatMemoryBlock serialises the relevant memory entries into a single
// system message that the model can interpret as structured past context.
// Entries are included greedily until the token budget is exhausted.
func formatMemoryBlock(memories []*outbound.StoredEmbedding, budget int) string {
	if len(memories) == 0 || budget <= 0 {
		return ""
	}

	header := "[RELEVANT MEMORY]\n"
	used := EstimateTokens(header)
	body := header

	for _, m := range memories {
		line := "- " + m.Content + "\n"
		cost := EstimateTokens(line)
		if used+cost > budget {
			break
		}
		body += line
		used += cost
	}

	if body == header {
		// Nothing fit inside the budget
		return ""
	}
	return body
}

// fitRecentMessages returns the most recent messages that fit within the token
// budget. When the full history exceeds the budget, the oldest messages are
// dropped first to preserve conversational continuity.
func fitRecentMessages(messages []outbound.Message, budget int) []outbound.ChatMessage {
	if len(messages) == 0 || budget <= 0 {
		return nil
	}

	// Walk backwards from the newest message, accumulate until budget exhausted.
	type indexed struct {
		idx int
		msg outbound.ChatMessage
	}
	var selected []indexed
	used := 0

	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		content := m.Role + ": " + m.Content
		cost := EstimateTokens(content)
		if used+cost > budget {
			break
		}
		selected = append(selected, indexed{i, outbound.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}})
		used += cost
	}

	// Reverse so messages are in chronological order (oldest first).
	result := make([]outbound.ChatMessage, len(selected))
	for i, s := range selected {
		result[len(selected)-1-i] = s.msg
	}
	return result
}

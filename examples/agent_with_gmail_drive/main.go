// Package main demonstrates a minimal working example of the DevilFish agent
// with Gmail and Google Drive MCP integration, skills, and the agent safety layer.
//
// This example uses mock implementations so it can run without:
// - An actual AI provider API key
// - Real Gmail or Google Drive MCP servers
//
// Run with: go run ./examples/agent_with_gmail_drive/
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"devilfish/internal/application/agent"
	"devilfish/internal/application/policy"
	"devilfish/internal/application/skill"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// mockLogger is a simple logger that prints to stdout.
type mockLogger struct{}

func (l *mockLogger) Debug(msg string)                   { fmt.Printf("  [DEBUG] %s\n", msg) }
func (l *mockLogger) Debugf(f string, a ...interface{})  { fmt.Printf("  [DEBUG] "+f+"\n", a...) }
func (l *mockLogger) Info(msg string)                    { fmt.Printf("  [INFO]  %s\n", msg) }
func (l *mockLogger) Infof(f string, a ...interface{})   { fmt.Printf("  [INFO]  "+f+"\n", a...) }
func (l *mockLogger) Warn(msg string)                    { fmt.Printf("  [WARN]  %s\n", msg) }
func (l *mockLogger) Warnf(f string, a ...interface{})   { fmt.Printf("  [WARN]  "+f+"\n", a...) }
func (l *mockLogger) Error(msg string)                   { fmt.Printf("  [ERROR] %s\n", msg) }
func (l *mockLogger) Errorf(f string, a ...interface{})  { fmt.Printf("  [ERROR] "+f+"\n", a...) }
func (l *mockLogger) With(fields map[string]interface{}) logging.Logger {
	return l
}

// mockGmailClient implements both outbound.MCPClient and outbound.EmailCapability.
// This allows the skill registry to store it as MCPClient and the Gmail skills
// to type-assert it to EmailCapability.
type mockGmailClient struct {
	mu        sync.Mutex
	connected bool
	tools     []outbound.MCPTool
}

func newMockGmailClient() *mockGmailClient {
	return &mockGmailClient{
		tools: []outbound.MCPTool{
			{
				Name:        "SendEmail",
				Description: "Send an email via Gmail",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"to":      map[string]interface{}{"type": "string"},
						"subject": map[string]interface{}{"type": "string"},
						"body":    map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				Name:        "ListEmails",
				Description: "List emails matching a query",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				Name:        "ReadEmail",
				Description: "Read an email by ID",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
}

func (m *mockGmailClient) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockGmailClient) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockGmailClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockGmailClient) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	return m.tools, nil
}

func (m *mockGmailClient) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to Gmail MCP server")
	}

	fmt.Printf("    -> Gmail MCP executing tool: %s\n", toolName)

	switch toolName {
	case "SendEmail":
		to := getString(params, "to", "unknown")
		fmt.Printf("    -> Sending email to: %s\n", to)
		return map[string]interface{}{"status": "sent", "to": to}, nil

	case "ListEmails":
		query := getString(params, "query", "")
		fmt.Printf("    -> Listing emails with query: %q\n", query)
		return map[string]interface{}{
			"emails": []interface{}{
				map[string]interface{}{
					"id":      "msg-001",
					"from":    "boss@company.com",
					"to":      "me@gmail.com",
					"subject": "Q4 Report Deadline",
					"body":    "Please submit the Q4 report by Friday.",
				},
				map[string]interface{}{
					"id":      "msg-002",
					"from":    "newsletter@tech.com",
					"to":      "me@gmail.com",
					"subject": "Weekly Tech Digest",
					"body":    "Top stories this week in technology...",
				},
			},
		}, nil

	case "ReadEmail":
		id := getString(params, "id", "")
		fmt.Printf("    -> Reading email: %s\n", id)
		return map[string]interface{}{
			"email": map[string]interface{}{
				"id":      id,
				"from":    "boss@company.com",
				"to":      "me@gmail.com",
				"subject": "Q4 Report Deadline",
				"body":    "Hi,\n\nPlease submit the Q4 report by end of day Friday.\n\nThanks,\nBoss",
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown Gmail tool: %s", toolName)
	}
}

func (m *mockGmailClient) ServerName() string {
	return "gmail"
}

// EmailCapability methods (outbound.EmailCapability interface)

func (m *mockGmailClient) SendEmail(ctx context.Context, to, subject, body string) error {
	_, err := m.ExecuteTool(ctx, "SendEmail", map[string]interface{}{
		"to": to, "subject": subject, "body": body,
	})
	return err
}

func (m *mockGmailClient) ListEmails(ctx context.Context, query string) ([]outbound.Email, error) {
	result, err := m.ExecuteTool(ctx, "ListEmails", map[string]interface{}{"query": query})
	if err != nil {
		return nil, err
	}

	emailsRaw, ok := result["emails"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid email list format")
	}

	emails := make([]outbound.Email, 0, len(emailsRaw))
	for _, raw := range emailsRaw {
		mm, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		emails = append(emails, outbound.Email{
			ID:      getString(mm, "id", ""),
			From:    getString(mm, "from", ""),
			Subject: getString(mm, "subject", ""),
			Snippet: getString(mm, "body", ""),
			Body:    getString(mm, "body", ""),
		})
	}
	return emails, nil
}

func (m *mockGmailClient) ReadEmail(ctx context.Context, id string) (outbound.Email, error) {
	result, err := m.ExecuteTool(ctx, "ReadEmail", map[string]interface{}{"id": id})
	if err != nil {
		return outbound.Email{}, err
	}

	emailRaw, ok := result["email"].(map[string]interface{})
	if !ok {
		return outbound.Email{}, fmt.Errorf("invalid email format")
	}

	return outbound.Email{
		ID:      getString(emailRaw, "id", ""),
		From:    getString(emailRaw, "from", ""),
		Subject: getString(emailRaw, "subject", ""),
		Body:    getString(emailRaw, "body", ""),
	}, nil
}

// mockDriveClient implements both outbound.MCPClient and outbound.DriveCapability.
type mockDriveClient struct {
	mu        sync.Mutex
	connected bool
	tools     []outbound.MCPTool
}

func newMockDriveClient() *mockDriveClient {
	return &mockDriveClient{
		tools: []outbound.MCPTool{
			{
				Name:        "list_files",
				Description: "List files in Google Drive",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				Name:        "read_file",
				Description: "Read a file from Google Drive",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_id": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
}

func (m *mockDriveClient) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockDriveClient) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockDriveClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockDriveClient) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	return m.tools, nil
}

func (m *mockDriveClient) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to Google Drive MCP server")
	}

	fmt.Printf("    -> Drive MCP executing tool: %s\n", toolName)

	switch toolName {
	case "list_files":
		query := getString(params, "query", "")
		fmt.Printf("    -> Listing Drive files with query: %q\n", query)
		return map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"id":        "file-abc123",
					"name":      "report.pdf",
					"mime_type": "application/pdf",
					"size":      int64(2048576),
				},
				map[string]interface{}{
					"id":        "file-def456",
					"name":      "meeting_notes.docx",
					"mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
					"size":      int64(51200),
				},
			},
		}, nil

	case "read_file":
		fileID := getString(params, "file_id", "")
		fmt.Printf("    -> Reading Drive file: %s\n", fileID)
		return map[string]interface{}{
			"content": "This is the content of report.pdf.\n\nQ4 Financial Report\n====================\n\nRevenue: $1.2M\nExpenses: $800K\nProfit: $400K\n\nGrowth: 15% YoY",
		}, nil

	default:
		return nil, fmt.Errorf("unknown Drive tool: %s", toolName)
	}
}

func (m *mockDriveClient) ServerName() string {
	return "google-drive"
}

// DriveCapability methods (outbound.DriveCapability interface)

func (m *mockDriveClient) ListFiles(ctx context.Context, query string) ([]outbound.File, error) {
	result, err := m.ExecuteTool(ctx, "list_files", map[string]interface{}{"query": query})
	if err != nil {
		return nil, err
	}

	filesRaw, ok := result["files"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid file list format")
	}

	files := make([]outbound.File, 0, len(filesRaw))
	for _, raw := range filesRaw {
		mm, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		files = append(files, outbound.File{
			ID:       getString(mm, "id", ""),
			Name:     getString(mm, "name", ""),
			MIMEType: getString(mm, "mime_type", ""),
			Size:     getInt64(mm, "size", 0),
		})
	}
	return files, nil
}

func (m *mockDriveClient) ReadFile(ctx context.Context, fileID string) (string, error) {
	result, err := m.ExecuteTool(ctx, "read_file", map[string]interface{}{"file_id": fileID})
	if err != nil {
		return "", err
	}
	return getString(result, "content", ""), nil
}

// mockAIProvider simulates an AI provider with deterministic responses.
// It returns a scripted sequence of responses to demonstrate the agent loop.
type mockAIProvider struct {
	responses []string
	index     int
	mu        sync.Mutex
}

func newMockAIProvider(responses []string) *mockAIProvider {
	return &mockAIProvider{responses: responses}
}

func (m *mockAIProvider) Chat(ctx context.Context, req *outbound.ChatRequest) (*outbound.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index >= len(m.responses) {
		// If we run out of scripted responses, return a final answer
		return &outbound.ChatResponse{
			Content: "I have completed all the requested actions. The report.pdf file content has been read and the email has been sent to john@example.com.",
			Model:   "mock-llm",
		}, nil
	}

	resp := m.responses[m.index]
	m.index++

	fmt.Printf("  [LLM] Response #%d:\n%s\n", m.index, indent(resp, 4))

	return &outbound.ChatResponse{
		Content: resp,
		Model:   "mock-llm",
	}, nil
}

func (m *mockAIProvider) StreamChat(ctx context.Context, req *outbound.ChatRequest, onChunk func(string)) error {
	resp, err := m.Chat(ctx, req)
	if err != nil {
		return err
	}
	// Simulate streaming by sending the whole content as one chunk
	onChunk(resp.Content)
	return nil
}

func (m *mockAIProvider) IsAvailable() bool { return true }
func (m *mockAIProvider) Name() string      { return "mock-llm" }

// ============================================================================
// Helper Functions
// ============================================================================

func getString(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getInt64(m map[string]interface{}, key string, defaultVal int64) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case float64:
			return int64(val)
		case int:
			return int64(val)
		}
	}
	return defaultVal
}

func indent(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// printBanner prints a section divider.
func printBanner(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("  %s\n", title)
	fmt.Println(strings.Repeat("=", 70))
}

// ============================================================================
// Main
// ============================================================================

func main() {
	ctx := context.Background()

	printBanner("DevilFish - Agent with Gmail & Drive MCP Integration")

	// ========================================================================
	// STEP 1: Setup Logger
	// ========================================================================
	fmt.Println("\n[Step 1] Setting up logger...")
	logger := &mockLogger{}
	fmt.Println("  Logger created: mockLogger (prints to stdout)")

	// ========================================================================
	// STEP 2: Setup MCP Registry & Register Servers
	// ========================================================================
	printBanner("Step 2: MCP Registry Setup")

	// In production, the MCP Registry manages server connections and routes
	// tool calls to the appropriate server. For this example, we create
	// mock clients that implement both MCPClient and the capability interfaces
	// (EmailCapability, DriveCapability), which is how real adapters work.
	// The skills below use these clients directly to demonstrate the pattern.

	// Create mock Gmail MCP client (implements MCPClient + EmailCapability)
	gmailClient := newMockGmailClient()
	gmailConfig := outbound.MCPServerConfig{
		Name:      "gmail",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-gmail"},
	}

	// Connect Gmail client
	if err := gmailClient.Connect(ctx, gmailConfig); err != nil {
		fmt.Printf("  ERROR: Failed to connect Gmail: %v\n", err)
		return
	}
	fmt.Println("  Gmail MCP client connected")
	fmt.Println("  Gmail MCP server registered (mock)")

	// Create mock Drive MCP client (implements MCPClient + DriveCapability)
	driveClient := newMockDriveClient()
	driveConfig := outbound.MCPServerConfig{
		Name:      "google-drive",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-google-drive"},
	}

	// Connect Drive client
	if err := driveClient.Connect(ctx, driveConfig); err != nil {
		fmt.Printf("  ERROR: Failed to connect Drive: %v\n", err)
		return
	}
	fmt.Println("  Google Drive MCP client connected")
	fmt.Println("  Google Drive MCP server registered (mock)")

	fmt.Println("\n  Registered servers: gmail, google-drive")

	// ========================================================================
	// STEP 3: Setup Skill Registry & Register Skills
	// ========================================================================
	printBanner("Step 3: Skill Registry Setup")

	// Create skill registry
	skillRegistry := skill.NewRegistry()

	// Register Gmail skills
	// Note: The real skills use registry.Get() to find MCP clients and
	// type-assert to capability interfaces. For this example, we create
	// simplified mock skills that demonstrate the same pattern.

	// SendEmail skill
	sendEmailSkill := &mockSendEmailSkill{gmailClient: gmailClient}
	skillRegistry.Register(sendEmailSkill)
	fmt.Println("  Registered skill: send_email")

	// ListEmails skill
	listEmailsSkill := &mockListEmailsSkill{gmailClient: gmailClient}
	skillRegistry.Register(listEmailsSkill)
	fmt.Println("  Registered skill: list_emails")

	// ReadEmail skill
	readEmailSkill := &mockReadEmailSkill{gmailClient: gmailClient}
	skillRegistry.Register(readEmailSkill)
	fmt.Println("  Registered skill: read_email")

	// ListFiles skill
	listFilesSkill := &mockListFilesSkill{driveClient: driveClient}
	skillRegistry.Register(listFilesSkill)
	fmt.Println("  Registered skill: list_files")

	// ReadFile skill
	readFileSkill := &mockReadFileSkill{driveClient: driveClient}
	skillRegistry.Register(readFileSkill)
	fmt.Println("  Registered skill: read_file")

	fmt.Printf("\n  Total skills registered: %d\n", skillRegistry.Count())

	// ========================================================================
	// STEP 4: Setup Execution Policies (Agent Safety Layer)
	// ========================================================================
	printBanner("Step 4: Execution Policy Setup")

	policySet := policy.NewPolicySet()

	// Policy 1: Email confirmation required for send_email
	emailPolicy := policy.NewEmailPolicy()
	emailPolicy.RequireConfirmation = true
	policySet.Add(emailPolicy)
	fmt.Println("  Policy added: EmailPolicy (require confirmation for send_email)")

	// Policy 2: Rate limiting - max 10 calls per minute per tool
	rateLimitPolicy := policy.NewRateLimitPolicy(10, 1*time.Minute)
	policySet.Add(rateLimitPolicy)
	fmt.Println("  Policy added: RateLimitPolicy (max 10 calls/minute per tool)")

	// Policy 3: Allowed tools whitelist
	allowedToolsPolicy := policy.NewAllowedToolsPolicy([]string{
		"send_email", "list_emails", "read_email", "list_files", "read_file",
	})
	policySet.Add(allowedToolsPolicy)
	fmt.Println("  Policy added: AllowedToolsPolicy (whitelist)")

	fmt.Println("\n  Policies configured: 3")

	// ========================================================================
	// STEP 5: Setup Agent
	// ========================================================================
	printBanner("Step 5: Agent Setup")

	// Build system prompt with tool schemas
	systemPrompt := agent.BuildSystemPrompt(skillRegistry, "")
	fmt.Printf("  System prompt generated (%d bytes)\n", len(systemPrompt))

	// Create mock AI provider with scripted responses
	// These simulate what a real LLM would output for the user's request.
	mockProvider := newMockAIProvider([]string{
		// Iteration 1: Agent decides to list files first to find report.pdf
		`{
  "thought": "The user wants to send report.pdf to john@example.com. First, I need to find the file in Google Drive to get its ID and read its content.",
  "action": "list_files",
  "action_input": {"query": "name contains 'report'"}
}`,
		// Iteration 2: Agent reads the file content
		`{
  "thought": "Found report.pdf with ID file-abc123. Now I'll read the file content to include in the email.",
  "action": "read_file",
  "action_input": {"file_id": "file-abc123"}
}`,
		// Iteration 3: Agent sends the email WITH confirmation (policy requirement)
		`{
  "thought": "I have the file content. Now I'll compose and send the email to john@example.com with the report attached. I'm setting confirmed=true as required by the email policy.",
  "action": "send_email",
  "action_input": {
    "to": "john@example.com",
    "subject": "Q4 Report - report.pdf",
    "body": "Hi John,\n\nPlease find the Q4 Report attached.\n\nKey highlights:\n- Revenue: $1.2M\n- Expenses: $800K\n- Profit: $400K\n- Growth: 15% YoY\n\nBest regards",
    "confirmed": true
  }
}`,
		// Iteration 4: Final answer (no tool calls)
		`{
  "thought": "I have completed all the requested actions successfully.",
  "final_answer": "Done! I found report.pdf in your Google Drive, read its content, and sent an email to john@example.com with the Q4 Report highlights included in the body."
}`,
	})
	fmt.Println("  Mock AI provider created with 4 scripted responses")

	// Create agent with configuration
	agentConfig := agent.Config{
		MaxIterations: 10,
		SystemPrompt:  systemPrompt,
		Model:         "mock-llm",
		Temperature:   0.7,
	}

	ag := agent.NewAgent(agentConfig, skillRegistry, policySet, mockProvider, logger)
	fmt.Printf("  Agent created: max_iterations=%d, model=%s\n", agentConfig.MaxIterations, agentConfig.Model)

	// ========================================================================
	// STEP 6: Run Agent
	// ========================================================================
	printBanner("Step 6: Running Agent")

	// Simulate user input
	userInput := "Send the file report.pdf to john@example.com"
	fmt.Printf("\n  User input: %q\n\n", userInput)

	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  AGENT EXECUTION TRACE")
	fmt.Println(strings.Repeat("-", 70))

	// Run the agent
	startTime := time.Now()
	response, err := ag.Run(ctx, userInput)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("\n  ERROR: Agent failed: %v\n", err)
		return
	}

	// ========================================================================
	// STEP 7: Print Results & Observability
	// ========================================================================
	printBanner("Step 7: Results & Observability")

	// Print final response
	fmt.Println("\n  Final Agent Response:")
	fmt.Println(strings.Repeat("  -", 34))
	for _, line := range strings.Split(response, "\n") {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println(strings.Repeat("  -", 34))

	// Print execution steps
	steps := ag.GetSteps()
	fmt.Printf("\n  Execution Steps: %d total\n", len(steps))
	fmt.Println()

	for i, step := range steps {
		fmt.Printf("  Step %d: [%s]\n", i+1, step.Duration.Round(time.Millisecond))

		if step.Thought != "" {
			// Extract thought from JSON or text
			thought := step.Thought
			if len(thought) > 100 {
				thought = thought[:100] + "..."
			}
			fmt.Printf("    Thought: %s\n", thought)
		}

		if step.ToolCall != nil {
			fmt.Printf("    Tool: %s\n", step.ToolCall.Name)
			fmt.Printf("    Params: %v\n", step.ToolCall.Parameters)
		}

		if step.Result != "" {
			result := step.Result
			if len(result) > 80 {
				result = result[:80] + "..."
			}
			fmt.Printf("    Result: %s\n", result)
		}

		if step.Error != nil {
			fmt.Printf("    Error: %v\n", step.Error)
		}

		fmt.Println()
	}

	// Summary
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("  Total time: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Steps executed: %d\n", len(steps))
	fmt.Printf("  Tools called: %d\n", countToolCalls(steps))
	fmt.Printf("  Policy denials: %d\n", countPolicyDenials(steps))
	fmt.Println()

	// Demonstrate policy denial
	printBanner("Bonus: Policy Denial Demo")
	fmt.Println("\n  Testing policy: trying to send email WITHOUT confirmation...")

	deniedCall := skill.ToolCall{
		Name: "send_email",
		Parameters: map[string]interface{}{
			"to":      "spammer@evil.com",
			"subject": "Buy now!",
			"body":    "Spam content",
			// Note: "confirmed" is NOT set
		},
	}

	if err := policySet.Allow(deniedCall); err != nil {
		fmt.Printf("  POLICY DENIED: %v\n", err)
	} else {
		fmt.Println("  Policy allowed (unexpected)")
	}

	fmt.Println("\n  Testing policy: trying blocked recipient...")
	emailPolicy.BlockedRecipients = append(emailPolicy.BlockedRecipients, "spammer@evil.com")
	blockedCall := skill.ToolCall{
		Name: "send_email",
		Parameters: map[string]interface{}{
			"to":        "spammer@evil.com",
			"subject":   "Hello",
			"body":      "Hi",
			"confirmed": true,
		},
	}

	if err := policySet.Allow(blockedCall); err != nil {
		fmt.Printf("  POLICY DENIED: %v\n", err)
	} else {
		fmt.Println("  Policy allowed (unexpected)")
	}

	fmt.Println("\n  Testing policy: allowed recipient with confirmation...")
	allowedCall := skill.ToolCall{
		Name: "send_email",
		Parameters: map[string]interface{}{
			"to":        "john@example.com",
			"subject":   "Hello",
			"body":      "Hi",
			"confirmed": true,
		},
	}

	if err := policySet.Allow(allowedCall); err != nil {
		fmt.Printf("  POLICY DENIED: %v\n", err)
	} else {
		fmt.Println("  Policy ALLOWED: email sending permitted")
	}

	printBanner("Example Complete")
	fmt.Println("\n  This example demonstrated:")
	fmt.Println("  1. MCP Registry with Gmail and Drive servers (mock)")
	fmt.Println("  2. Skill Registry with 5 skills (Gmail + Drive)")
	fmt.Println("  3. Execution Policies (email confirmation, rate limit, whitelist)")
	fmt.Println("  4. Agent loop with LLM simulation")
	fmt.Println("  5. Tool execution through skills")
	fmt.Println("  6. Policy enforcement (denials)")
	fmt.Println("  7. Observability (steps, timing, trace)")
	fmt.Println()
}

// ============================================================================
// Mock Skill Implementations
// ============================================================================
// These are simplified versions of the real skills that work with the mock clients.
// They demonstrate the same pattern: validate args -> get client -> execute -> return result.

// mockSendEmailSkill sends an email via Gmail.
type mockSendEmailSkill struct {
	gmailClient *mockGmailClient
}

func (s *mockSendEmailSkill) Name() string        { return "send_email" }
func (s *mockSendEmailSkill) Description() string { return "Send an email via Gmail MCP. Requires 'to', 'subject', and 'body' parameters." }
func (s *mockSendEmailSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"to":      map[string]interface{}{"type": "string", "description": "Recipient email address"},
			"subject": map[string]interface{}{"type": "string", "description": "Email subject line"},
			"body":    map[string]interface{}{"type": "string", "description": "Email body content"},
		},
		"required": []string{"to", "subject", "body"},
	}
}

func (s *mockSendEmailSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	to, ok := args["to"].(string)
	if !ok || to == "" {
		return "", fmt.Errorf("invalid or missing 'to' parameter")
	}
	subject, _ := args["subject"].(string)
	body, _ := args["body"].(string)

	if err := s.gmailClient.SendEmail(ctx, to, subject, body); err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}
	return fmt.Sprintf("Email sent successfully to %s", to), nil
}

// mockListEmailsSkill lists emails from Gmail.
type mockListEmailsSkill struct {
	gmailClient *mockGmailClient
}

func (s *mockListEmailsSkill) Name() string        { return "list_emails" }
func (s *mockListEmailsSkill) Description() string { return "List emails in Gmail matching a query. Uses Gmail search syntax." }
func (s *mockListEmailsSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Gmail search query"},
		},
		"required": []string{"query"},
	}
}

func (s *mockListEmailsSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		query = "in:inbox"
	}

	emails, err := s.gmailClient.ListEmails(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to list emails: %w", err)
	}

	if len(emails) == 0 {
		return "No emails found.", nil
	}

	result := fmt.Sprintf("Found %d email(s):\n", len(emails))
	for _, email := range emails {
		result += fmt.Sprintf("- ID: %s | From: %s | Subject: %s\n", email.ID, email.From, email.Subject)
	}
	return result, nil
}

// mockReadEmailSkill reads a specific email.
type mockReadEmailSkill struct {
	gmailClient *mockGmailClient
}

func (s *mockReadEmailSkill) Name() string        { return "read_email" }
func (s *mockReadEmailSkill) Description() string { return "Read a specific email by its ID. Returns the full email content." }
func (s *mockReadEmailSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string", "description": "Email ID to read"},
		},
		"required": []string{"id"},
	}
}

func (s *mockReadEmailSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("invalid or missing 'id' parameter")
	}

	email, err := s.gmailClient.ReadEmail(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}

	result := fmt.Sprintf("Email Details:\nID: %s\nFrom: %s\nTo: %s\nSubject: %s\nBody:\n%s",
		email.ID, email.From, email.To, email.Subject, email.Body)
	return result, nil
}

// mockListFilesSkill lists files from Google Drive.
type mockListFilesSkill struct {
	driveClient *mockDriveClient
}

func (s *mockListFilesSkill) Name() string        { return "list_files" }
func (s *mockListFilesSkill) Description() string { return "List files in Google Drive. Supports Google Drive search syntax." }
func (s *mockListFilesSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Google Drive search query"},
		},
		"required": []string{"query"},
	}
}

func (s *mockListFilesSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)

	files, err := s.driveClient.ListFiles(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return "No files found.", nil
	}

	result := fmt.Sprintf("Found %d file(s):\n", len(files))
	for _, file := range files {
		result += fmt.Sprintf("- ID: %s | Name: %s | Type: %s\n", file.ID, file.Name, file.MIMEType)
	}
	return result, nil
}

// mockReadFileSkill reads a file from Google Drive.
type mockReadFileSkill struct {
	driveClient *mockDriveClient
}

func (s *mockReadFileSkill) Name() string        { return "read_file" }
func (s *mockReadFileSkill) Description() string { return "Read the content of a file from Google Drive by its ID." }
func (s *mockReadFileSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_id": map[string]interface{}{"type": "string", "description": "Google Drive file ID"},
		},
		"required": []string{"file_id"},
	}
}

func (s *mockReadFileSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	fileID, ok := args["file_id"].(string)
	if !ok || fileID == "" {
		return "", fmt.Errorf("invalid or missing 'file_id' parameter")
	}

	content, err := s.driveClient.ReadFile(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("File Content:\n\n%s", content), nil
}

// ============================================================================
// Utility Functions
// ============================================================================

func countToolCalls(steps []agent.Step) int {
	count := 0
	for _, step := range steps {
		if step.ToolCall != nil {
			count++
		}
	}
	return count
}

func countPolicyDenials(steps []agent.Step) int {
	count := 0
	for _, step := range steps {
		if step.Error != nil && strings.Contains(step.Error.Error(), "policy denied") {
			count++
		}
	}
	return count
}

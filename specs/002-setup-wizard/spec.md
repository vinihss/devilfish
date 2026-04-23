# Feature Specification: Setup Wizard CLI

**Feature Branch**: `002-setup-wizard`  
**Created**: 2026-04-23  
**Status**: Draft  
**Input**: User description: "Como usuario do sistema, necessito que no processo de instalação da aplicação, seja feito um passo a passo de configuração dos agentes, messenger e mcps. E que também tenha a opção de fazer configurações e ajustes via CLI."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Interactive Setup Wizard (Priority: P1)

As a new user of the system, I need to run an interactive setup wizard during installation so that I can configure all system components step by step without manual editing of configuration files.

**Why this priority**: New users need a guided experience to set up the system correctly without reading documentation or manually editing YAML files.

**Independent Test**: Can run `./devilfish setup` and complete all configuration steps interactively.

**Acceptance Scenarios**:

1. **Given** a fresh installation with no config, **When** user runs `./devilfish setup`, **Then** wizard prompts for each component (AI providers, messaging channels, MCP servers)
2. **Given** existing config, **When** user runs setup, **Then** wizard loads current config and offers to modify
3. **Given** partial config, **When** wizard runs, **Then** wizard only prompts for missing required items

---

### User Story 2 - Configure AI Providers (Priority: P1)

As a user, I need to configure AI providers during setup so that the system can connect to AI services.

**Why this priority**: AI providers are the core functionality of the system.

**Independent Test**: Can configure OpenAI, Groq, Gemini, Ollama through wizard.

**Acceptance Scenarios**:

1. **Given** wizard prompt for AI, **When** user selects provider and enters API key, **Then** provider is configured and saved
2. **Given** multiple providers, **When** user configures them, **Then** all providers are saved
3. **Given** invalid API key, **When** user enters, **Then** wizard validates and shows error with retry option

---

### User Story 3 - Configure Messaging Channels (Priority: P1)

As a user, I need to configure messaging channels during setup so that the system can receive messages from Telegram, Discord, or Slack.

**Why this priority**: Messaging channels are how users interact with the system.

**Independent Test**: Can configure Telegram, Discord, Slack through wizard.

**Acceptance Scenarios**:

1. **Given** wizard prompt for messaging, **When** user enables channel and enters token, **Then** channel is configured
2. **Given** channel configuration saved, **When** wizard runs again, **Then** current config is displayed for review/modification
3. **Given** invalid token, **When** user enters, **Then** wizard shows error with retry option

---

### User Story 4 - Configure MCP Servers (Priority: P2)

As a user, I need to configure MCP servers during setup so that the system has access to external tools.

**Why this priority**: MCP servers extend system capabilities with additional tools.

**Independent Test**: Can configure filesystem, github, memory MCP servers through wizard.

**Acceptance Scenarios**:

1. **Given** wizard prompt for MCP, **When** user selects server type, **Then** appropriate prompts are shown based on transport
2. **Given** stdio transport, **When** user configures, **Then** command/args are collected
3. **Given** http transport, **When** user configures, **Then** URL and auth token are collected

---

### User Story 5 - CLI Commands for Configuration (Priority: P2)

As a user, I need CLI commands to view and modify configuration so that I can make adjustments without the wizard.

**Why this priority**: Power users prefer CLI commands for quick changes.

**Independent Test**: Can run config get/set commands.

**Acceptance Scenarios**:

1. **Given** `config show` command, **When** user runs, **Then** current config is displayed
2. **Given** `config get <key>` command, **When** user runs, **Then** value is returned
3. **Given** `config set <key> <value>` command, **When** user runs, **Then** config is updated
4. **Given** `config validate` command, **When** user runs, **Then** config is validated

---

### Edge Cases

- What happens when required API keys are missing at startup?
- How does the system handle invalid configuration values?
- What happens when network is unavailable during validation?
- How are secrets handled in CLI output (should be masked)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide interactive setup wizard via `setup` command
- **FR-002**: Setup wizard MUST validate all input before saving
- **FR-003**: Setup wizard MUST support skip/retry on validation errors
- **FR-004**: System MUST load existing config when running setup wizard
- **FR-005**: Setup wizard MUST support configuring AI providers (OpenAI, Groq, Gemini, Ollama)
- **FR-006**: Setup wizard MUST support configuring messaging channels (Telegram, Discord, Slack)
- **FR-007**: Setup wizard MUST support configuring MCP servers
- **FR-008**: CLI MUST provide `config show` command to display current configuration
- **FR-009**: CLI MUST provide `config get <key>` command to get specific value
- **FR-010**: CLI MUST provide `config set <key> <value>` command to update config
- **FR-011**: CLI MUST mask secrets when displaying configuration
- **FR-012**: CLI MUST provide `config validate` command to verify configuration
- **FR-013**: System MUST fail gracefully if required config is missing

### Key Entities *(include if feature involves data)*

- **Configuration**: All system settings (AI, messaging, MCP, security)
- **Provider**: AI provider configuration (name, API key, model)
- **Channel**: Messaging channel configuration (name, token, enabled)
- **MCPServer**: MCP server configuration

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Setup wizard completes in under 5 minutes for full configuration
- **SC-002**: CLI commands respond in under 1 second
- **SC-003**: 100% of configuration values can be viewed via CLI
- **SC-004**: Configuration validation catches 100% of invalid values
- **SC-005**: Secrets are never displayed in plain text in CLI output

## Assumptions

- CLI will be the primary entry point (alongside server)
- Configuration file uses YAML format (already defined)
- Secrets are stored in environment variables, not in config file
- Wizard uses terminal-compatible prompts (compatible with common terminals)
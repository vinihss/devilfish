# Tasks: Setup Wizard CLI

**Input**: Design documents from `/specs/002-setup-wizard/`
**Prerequisites**: spec.md (required), plan.md (required)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: CLI infrastructure and foundation

- [ ] T001 Create CLI entry point in cmd/cli/main.go with cobra
- [ ] T002 [P] Create config loader utility in internal/infra/config/cli.go
- [ ] T003 Add cobra to go.mod dependencies

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Define configuration struct for CLI in internal/infra/config/cli_types.go
- [ ] T005 Implement config validation in internal/infra/config/validator.go
- [ ] T006 Add secret masking utility

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Setup Wizard Core (Priority: P1) 🎯 MVP

**Goal**: Interactive setup wizard that guides users through configuration

**Independent Test**: Run `./devilfish setup` and complete all prompts

### Implementation for User Story 1

- [ ] T007 [P] [US1] Create setup command in cmd/cli/setup.go
- [ ] T008 [P] [US1] Create interactive prompt helper in internal/infra/cli/prompts.go
- [ ] T009 [US1] Implement wizard flow controller
- [ ] T010 [US1] Add config save/load functionality

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - AI Provider Configuration (Priority: P1)

**Goal**: Configure AI providers through wizard

**Independent Test**: Configure OpenAI, Groq, Gemini, Ollama through wizard

### Implementation for User Story 2

- [ ] T011 [P] [US2] Create AI provider prompt in cmd/cli/setup/providers.go
- [ ] T012 [US2] Implement provider selection flow
- [ ] T013 [US2] Add API key input with validation display

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Messaging Channel Configuration (Priority: P1)

**Goal**: Configure messaging channels through wizard

**Independent Test**: Configure Telegram, Discord, Slack through wizard

### Implementation for User Story 3

- [ ] T014 [P] [US3] Create messaging channel prompt in cmd/cli/setup/channels.go
- [ ] T015 [US3] Implement channel selection flow
- [ ] T016 [US3] Add token input with validation display

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should work independently

---

## Phase 6: User Story 4 - MCP Server Configuration (Priority: P2)

**Goal**: Configure MCP servers through wizard

**Independent Test**: Configure filesystem, github, memory MCP servers through wizard

### Implementation for User Story 4

- [ ] T017 [P] [US4] Create MCP server prompt in cmd/cli/setup/mcp.go
- [ ] T018 [US4] Implement server type selection (stdio vs http)
- [ ] T019 [US4] Add command/URL input flow

**Checkpoint**: All user stories should now be independently functional

---

## Phase 7: User Story 5 - CLI Config Commands (Priority: P2)

**Goal**: CLI commands for viewing and modifying configuration

**Independent Test**: Run config get/set commands

### Implementation for User Story 5

- [ ] T020 [P] [US5] Implement config show command
- [ ] T021 [P] [US5] Implement config get command
- [ ] T022 [US5] Implement config set command
- [ ] T023 [US5] Implement config validate command
- [ ] T024 Implement secret masking in all outputs

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T025 Add help text for all commands
- [ ] T026 Add examples for config commands
- [ ] T027 Add error handling improvements
- [ ] T028 Update README with CLI documentation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)

### Within Each User Story

- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All prompt components can be developed in parallel
- Once Foundational phase completes, all user stories can start in parallel

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: Setup Wizard Core
4. **STOP and VALIDATE**: Test setup wizard
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add Setup Wizard Core → Test → Deploy (MVP!)
3. Add AI Provider Config → Test → Deploy
4. Add Messaging Channel Config → Test → Deploy
5. Add MCP Server Config → Test → Deploy
6. Add CLI Config Commands → Test → Deploy

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
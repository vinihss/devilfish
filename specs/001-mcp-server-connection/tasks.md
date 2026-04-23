# Tasks: MCP Server Connection

**Input**: Design documents from `/specs/001-mcp-server-connection/`
**Prerequisites**: spec.md (required)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and MCP adapter structure

- [X] T001 Create MCP adapter directory structure in internal/adapters/mcp/
- [X] T002 [P] Create MCP client interface in internal/ports/outbound/mcp_client.go
- [X] T003 [P] Add MCP configuration to configs/config.yaml (server list)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create MCP Server entity in internal/domain/entity/mcp_server.go
- [X] T005 Create Tool entity in internal/domain/entity/tool.go
- [X] T006 Create Connection value object in internal/domain/valueobject/mcp_connection.go
- [X] T007 Implement MCP client port interface in internal/ports/outbound/mcp_client.go
- [X] T008 Setup connection pool manager in internal/adapters/mcp/pool.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Connect to MCP Server (Priority: P1) 🎯 MVP

**Goal**: Establish connections to external MCP servers

**Independent Test**: Configure an MCP server endpoint and verify connection establishment

### Implementation for User Story 1

- [X] T009 [P] [US1] Implement stdio transport in internal/adapters/mcp/stdio_transport.go
- [X] T010 [P] [US1] Implement HTTP/SSE transport in internal/adapters/mcp/http_transport.go
- [X] T011 [US1] Implement MCP client adapter in internal/adapters/mcp/client.go
- [X] T012 [US1] Add connection establishment logic with authentication
- [X] T013 [US1] Handle connection errors and timeouts
- [X] T014 [US1] Add connection state management

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Execute Tools via MCP Server (Priority: P2)

**Goal**: Execute tools hosted on MCP servers

**Independent Test**: Call a tool on a connected MCP server and verify tool executes correctly

### Implementation for User Story 2

- [X] T015 [P] [US2] Implement tool discovery/listing in internal/adapters/mcp_tool_discovery.go
- [X] T016 [US2] Implement tool execution request/response handling
- [X] T017 [US2] Add tool input validation per schema
- [X] T018 [US2] Map tool responses back to caller
- [X] T019 [US2] Handle tool execution errors gracefully

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Manage Multiple MCP Server Connections (Priority: P3)

**Goal**: Manage connections to multiple MCP servers simultaneously

**Independent Test**: Connect to multiple MCP servers and verify requests are routed correctly

### Implementation for User Story 3

- [X] T020 [P] [US3] Implement server registry in internal/adapters/mcp/registry.go
- [X] T021 [US3] Implement tool routing to correct server
- [X] T022 [US3] Add conflict resolution for duplicate tool names
- [X] T023 [US3] Implement connection health monitoring

**Checkpoint**: All user stories should now be independently functional

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T024 Add reconnection logic for failed connections
- [X] T025 [P] Update config validation
- [ ] T026 Add logging for MCP operations
- [ ] T027 Security hardening (credential handling)
- [ ] T028 Documentation updates

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 for tool execution
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Depends on US1 for connection management

### Within Each User Story

- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Phase 3 implementation marked [P] can run in parallel
- Once Foundational phase completes, all user stories can start in parallel

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
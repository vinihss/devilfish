# DevilFish Constitution

## Core Principles

### I. Ports & Adapters Architecture
The system MUST use Hexagonal Architecture (Ports & Adapters):
- **Inbound Ports**: Interfaces the external world can call (MessageHandler, WebSocket)
- **Outbound Ports**: Interfaces the domain uses to communicate with external services (AIProvider, MessageSource)
- **Adapters**: Concrete implementations of port interfaces
- **Domain**: Core business logic isolated from infrastructure concerns
- **Rationale**: Enables testability, flexibility, and multiple implementations of the same interface

### II. Small Components (NON-NEGOTIABLE)
Every component MUST adhere to these limits:
- **Packages**: Max 15 files per package
- **Functions**: Max 50 lines per function
- **Structs**: Max 8 fields per struct
- **Parameters**: Max 4 parameters → use struct for parameters
- **Rationale**: Maintain readability, testability, and enforce single responsibility

### III. Test-First (NON-NEGOTIABLE)
A mandatory test pyramid MUST be maintained:
- **Unit tests**: 10x ratio (fastest, most numerous)
- **Integration tests**: 5x ratio (focus areas: library contracts, inter-service communication)
- **E2E tests**: 1x ratio (minimal, critical paths only)
- **Test naming**: Descriptive pattern: `TestSubject_Condition_ExpectedBehavior`
- **Rationale**: Regression protection and documentation through tests

### IV. Observability & Logging
The system MUST provide full observability:
- **Structured logging**: Use key-value pairs, never string concatenation
- **Error wrapping**: Always wrap errors with context using `fmt.Errorf("...: %w", err)`
- **Secrets protection**: Never log passwords, tokens, API keys; mask sensitive data
- **Metrics**: Prometheus metrics for messages processed, AI request duration
- **Rationale**: Debugging and monitoring production systems

### V. Security & Input Validation
The system MUST enforce security at every layer:
- **Secrets management**: All secrets via environment variables, never hardcoded
- **Input validation**: Validate all external input before processing
- **Rate limiting**: Token bucket algorithm per user/channel
- **WebSocket auth**: JWT-based authentication for WebSocket connections
- **Rationale**: Protect against attacks and data breaches

## Code Quality Standards

### Go Conventions
- **Interfaces**: Small and focused (< 5 methods)
- **Constructors**: Inject all dependencies via constructors
- **Error handling**: Custom errors with `errors.New`, wrapped with `fmt.Errorf`
- **DRY**: Extract duplicated code to helper functions or abstractions
- **Cohesion**: Related code MUST be kept together

### Refactoring Triggers
The following conditions MUST trigger refactoring:
| Signal | Action |
|--------|--------|
| 3+ duplications | Extract helper function |
| Function > 50 lines | Break into smaller functions |
| Struct > 8 fields | Break into smaller structs |
| Parameters > 4 | Use struct for parameters |
| Large switch/if-else chain | Use strategy pattern or map |

### Concurrency Patterns
- **Goroutines**: Always with context for cancellation
- **Channels**: Fixed buffer size, always closed properly
- **Mutex vs Channel**: Mutex for shared state; Channel for ownership/signaling
- **Worker pools**: Preferred for parallel processing

## Development Workflow

### Project Structure
```
devilfish/
├── cmd/                    # Entry points
├── internal/
│   ├── domain/             # Core business logic (entities, value objects)
│   ├── application/        # Use cases and DTOs
│   ├── ports/             # Interface definitions
│   ├── adapters/          # Concrete implementations
│   └── infra/            # Infrastructure concerns
├── pkg/                   # Reusable packages
├── configs/               # Configuration files
└── tests/                 # Test suites
```

### Docker & Hot-Reload
- **Dockerfile**: Multi-stage build for production
- **Dockerfile.dev**: Hot-reload with Air for development
- **docker-compose.yaml**: Full stack with secrets management
- **.air.toml**: Hot-reload configuration

### i18n Requirements
- **Supported locales**: en, pt, es, zh
- **Files**: TOML format per locale in `internal/infra/i18n/locales/`
- **Fallback**: English as default

## Governance

### Amendment Procedure
1. Amendments MUST be proposed via PR to this document
2. Changes MUST include rationale and migration plan
3. Breaking changes require MAJOR version bump

### Version Policy
- **MAJOR**: Backward-incompatible governance changes or principle removals
- **MINOR**: New principles or materially expanded guidance
- **PATCH**: Clarifications, wording, non-semantic refinements

### Compliance
- All PRs MUST verify compliance with these principles
- Complexity deviations MUST be justified in code review
- Tests MUST accompany all new functionality

**Version**: 1.0.0 | **Ratified**: 2026-04-20 | **Last Amended**: 2026-04-22
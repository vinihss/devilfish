# Implementation Plan: Setup Wizard CLI

**Feature**: Setup Wizard CLI  
**Created**: 2026-04-23  
**Status**: Ready for Implementation

## Technical Context

### Technology Stack

- **Language**: Go 1.22+
- **CLI Framework**: cobra (or native flag parsing)
- **Interactive Prompts**: survey or bubbletea
- **Config Format**: YAML (already in use)

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- `gopkg.in/yaml.v3` - YAML parsing (already in use)

### File Structure

```
cmd/
├── devilfishd/           # Server entry
│   └── main.go
└── cli/                 # NEW: CLI entry
    └── main.go
    ├── setup.go         # Setup wizard commands
    └── config.go       # Config management commands
```

### Component Design

1. **Setup Wizard** (`cmd/cli/setup.go`)
   - `runSetupCmd` - root command for setup
   - `runInteractiveWiz` - interactive wizard flow
   - AI provider prompts
   - Messaging channel prompts
   - MCP server prompts

2. **Config Commands** (`cmd/cli/config.go`)
   - `configShowCmd` - display all config
   - `configGetCmd` - get specific value
   - `configSetCmd` - set value
   - `configValidateCmd` - validate config

### Implementation Order

1. CLI entry point with cobra
2. Config loading utility
3. Setup wizard core
4. AI provider prompts
5. Messaging channel prompts
6. MCP server prompts
7. Config CLI commands
8. Validation utilities

### Integration Points

- Uses existing config.yaml format
- Reuses config loading from `internal/infra/config`
- Integrates with MCP adapters (existing)

### Unknowns (Defer to Implementation)

- [ ] Which interactive prompt library (survey vs bubbletea)
- [ ] How to validate API keys during setup
- [ ] Whether to use env vars or config file for secrets
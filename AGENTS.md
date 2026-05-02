# DevilFish - AGENTS.md

> **ATENÇÃO:** Este arquivo é a autoridade máxima para todas as decisões de desenvolvimento do projeto DevilFish. Qualquer implementação deve seguir estas regras.

---

## Identidade do Projeto

- **Nome:** DevilFish
- **Tipo:** Harness de mensageria que conecta canais de mensagens a modelos de AI via gateway WebSocket
- **Linguagem:** Go 1.22+
- **Docker:** Sim
- **Arquitetura:** Clean Architecture (Ports & Adapters)

---

## Estrutura de Diretórios

```
devilfish/
├── cmd/
│   ├── devilfishd/              # Ponto de entrada principal
│   │   └── main.go
│   └── cli/                   # Ferramenta CLI opcional
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/             # Entidades do domínio
│   │   ├── message.go
│   │   ├── session.go
│   │   ├── user.go
│   │   ├── config.go
│   │   ├── mcp_server.go       # Servidor MCP
│   │   └── tool.go            # Ferramenta MCP
│   │   └── valueobject/       # Value Objects
│   │       ├── messagecontent.go
│   │       ├── provider.go
│   │       └── mcp_connection.go # Conexão MCP
│   ├── application/
│   │   ├── usecase/          # Casos de uso
│   │   │   ├── context_assembler.go
│   │   │   ├── handle_message.go
│   │   │   ├── chat_with_ai.go
│   │   │   ├── manage_session.go
│   │   │   └── route_message.go
│   │   ├── skill/            # Skills (camada de abstração)
│   │   │   ├── types.go          # Skill interface, ToolCall, ToolResult
│   │   │   ├── registry.go       # Skill registry
│   │   │   ├── gmail_skills.go   # Gmail skills (send_email, list_emails, read_email)
│   │   │   ├── drive_skills.go   # Drive skills (list_files, read_file)
│   │   │   └── drive_skills_test.go
│   │   ├── agent/            # Agent loop (LLM-driven)
│   │   │   ├── types.go          # Step, Message, Config
│   │   │   ├── agent.go          # Agent loop com steps, policies
│   │   │   ├── system_prompt.go  # System prompt, tool schemas
│   │   │   └── agent_test.go
│   │   ├── policy/           # Execution policies (safety layer)
│   │   │   ├── types.go          # ExecutionPolicy interface, PolicySet
│   │   │   ├── email_policy.go   # Email safety policies
│   │   │   ├── rate_limit_policy.go # Rate limiting
│   │   │   ├── allowed_tools_policy.go # Whitelist/blacklist
│   │   │   └── policy_test.go
│   │   └── dto/              # Data Transfer Objects
│   │       ├── message_dto.go
│   │       └── ai_response_dto.go
│   ├── ports/
│   │   ├── inbound/          # Interfaces de entrada
│   │   │   ├── message_handler.go
│   │   │   └── websocket.go
│   │   └── outbound/         # Interfaces de saída
│   │       ├── ai_provider.go
│   │       ├── message_source.go
│   │       ├── session_store.go
│   │       ├── mcp_client.go      # MCPClient interface, MCPTool
│   │       ├── email_capability.go # EmailCapability interface
│   │       └── drive_capability.go # DriveCapability interface
│   ├── adapters/
│   │   ├── messaging/        # Adaptadores de mensageria
│   │   │   ├── telegram/
│   │   │   │   ├── adapter.go
│   │   │   │   └── handler.go
│   │   │   ├── discord/
│   │   │   │   ├── adapter.go
│   │   │   │   └── handler.go
│   │   │   └── slack/
│   │   │       ├── adapter.go
│   │   │       └── handler.go
│   │   ├── ai/               # Adaptadores de AI
│   │   │   ├── openai/
│   │   │   │   └── provider.go
│   │   │   ├── groq/
│   │   │   │   └── provider.go
│   │   │   ├── gemini/
│   │   │   │   └── provider.go
│   │   │   └── ollama/
│   │   │       └── provider.go
│   │   ├── mcp/              # Adaptadores MCP
│   │   │   ├── client.go          # Cliente MCP principal
│   │   │   ├── pool.go            # Pool de conexões
│   │   │   ├── registry.go        # Registro de servidores
│   │   │   ├── stdio_transport.go # Transport stdio
│   │   │   ├── http_transport.go  # Transport HTTP/SSE
│   │   │   ├── tool_discovery.go  # Descoberta de ferramentas
│   │   │   ├── gmail/            # Adaptador Gmail MCP
│   │   │   │   ├── adapter.go     # Adaptador Gmail com EmailCapability
│   │   │   │   ├── errors.go      # Erros específicos
│   │   │   │   └── adapter_test.go
│   │   │   └── drive/            # Adaptador Google Drive MCP
│   │   │       ├── adapter.go     # Adaptador Drive com DriveCapability
│   │   │       └── adapter_test.go
│   │   ├── websocket/
│   │   │   └── gateway.go
│   │   └── storage/
│   │       └── memory/
│   │           └── session_store.go
│   ├── infra/
│   │   ├── config/
│   │   │   ├── config.go
│   │   │   └── loader.go
│   │   ├── logging/
│   │   │   └── logger.go
│   │   ├── i18n/
│   │   │   ├── loader.go
│   │   │   ├── locales/
│   │   │   │   ├── en.toml
│   │   │   │   ├── pt.toml
│   │   │   │   ├── es.toml
│   │   │   │   └── zh.toml
│   │   │   └── translator.go
│   │   └── security/
│   │       ├── middleware.go
│   │       └── ratelimit.go
│   └── middleware/
│       └── logging.go
├── pkg/
│   └── utils/
│       ├── retry/
│       │   └── retry.go
│       └── buffer/
│           └── ringbuffer.go
├── configs/
│   ├── config.yaml
│   ├── docker-compose.yaml
│   └── .air.toml
├── tests/
│   ├── unit/
│   │   └── ...
│   └── integration/
│       └── ...
├── go.mod
├── go.sum
├── Dockerfile
├── Dockerfile.dev
├── Makefile
└── README.md
```

---

## Regras de Código

### 1. Estrutura e Organização

| Regra | Descrição |
|-------|-----------|
| **Pacotes small** | Máx 15 arquivos por pacote |
| **Nomes significativos** | Variáveis/expressões autoexplicativas |
| **Funções small** | Máx 50 linhas por função |
| **Structs small** | Máx 8 campos por struct |
| **DRY** | Evitar duplicação, usar abstrações |
| **Coesão** | Código relacionado junto |

### 2. Padrões Go

```go
// BOM: Interfaces pequenas e focadas
type MessageHandler interface {
    Handle(ctx context.Context, msg *entity.Message) error
}

// RUIM: Interface grande
type EverythingHandler interface {
    Handle(Message)
    Process(AI)
    Save(Session)
    Connect(WS)
    // ...
}

// BOM: Construtor com dependências claras
func NewService(
    provider ports.AIProvider,
    store ports.SessionStore,
    logger log.Logger,
) *Service {
    return &Service{
        provider: provider,
        store:    store,
        logger:   logger,
    }
}
```

### 3. Tratamento de Erros

```go
// BOM: Erros wrapados com contexto
if err != nil {
    return fmt.Errorf("failed to send message to %s: %w", channelID, err)
}

// BOM: Errors customizados
var ErrProviderNotAvailable = errors.New("AI provider not available")

func (s *Service) Chat(ctx context.Context, msg string) error {
    if !s.provider.IsAvailable() {
        return fmt.Errorf("%w: %s", ErrProviderNotAvailable, s.provider.Name())
    }
    // ...
}
```

### 4. Logging

```go
// BOM: Structured logging
s.logger.Info("handling message",
    "message_id", msg.ID,
    "user_id", msg.UserID,
    "channel", msg.Channel,
)

// RUIM: Strings concatenadas
s.logger.Info("handling message: " + msg.ID)
```

---

## Regras de Refatoração

### 1. Quando Refatorar

| Sinal | Ação |
|-------|------|
| 3+ duplicações | Extrair função auxiliar |
| Função > 50 linhas | Quebrar em funções menores |
| Struct > 8 campos | Quebrar em structs menores |
| Parâmetros > 4 | Usar struct para parâmetros |
| switch grande | Usar strategy/map |
| Many if-else chain | Usar polymorphism |

### 2. Extraindo Código

```go
// ANTES (duplicado em múltiplos lugares)
if msg.Type == "text" {
    // processar texto
} else if msg.Type == "image" {
    // processar imagem
} else if msg.Type == "audio" {
    // processar áudio
}

// DEPOIS (extraído para strategy)
type MessageProcessor interface {
    Process(ctx context.Context, msg *entity.Message) error
}

var processors = map[string]MessageProcessor{
    "text":  &TextProcessor{},
    "image": &ImageProcessor{},
    "audio": &AudioProcessor{},
}

func ProcessMessage(msg *entity.Message) error {
    processor, ok := processors[msg.Type]
    if !ok {
        return ErrUnknownMessageType
    }
    return processor.Process(ctx, msg)
}
```

---

## Regras de Segurança

### 1. Configuração Sensível

#### 1.1 Variáveis de Ambiente

```yaml
# configs/config.yaml
ai:
  openai:
    api_key: ${OPENAI_API_KEY}  # BOM
    # api_key: "sk-xxx"       # RUIM: hardcoded

messaging:
  telegram:
    bot_token: ${TELEGRAM_BOT_TOKEN}
```

#### 1.2 Secrets no Docker

```yaml
# docker-compose.yaml
services:
  devilfishd:
    secrets:
      - openai_api_key

secrets:
  openai_api_key:
    external: true
```

### 2. Rate Limiting

```go
// internal/infra/security/ratelimit.go
type RateLimiter struct {
    requests map[string]*tokenBucket
    mu       sync.Mutex
}

type tokenBucket struct {
    tokens    int
    maxTokens int
    refillAt  time.Time
}

func (r *RateLimiter) Allow(key string, cost int) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    bucket := r.requests[key]
    if bucket == nil {
        bucket = newTokenBucket(defaultMaxTokens)
        r.requests[key] = bucket
    }

    if time.Now().Before(bucket.refillAt) {
        return bucket.tokens >= cost
    }
    bucket.refill()
    return bucket.tokens >= cost
}
```

### 3. Validação de Entrada

```go
// BOM: Validar toda entrada
type Message struct {
    Content string `json:"content" validate:"required,min=1,max=4000"`
    UserID  string `json:"user_id" validate:"required,uuid4"`
    Channel string `json:"channel" validate:"required,oneof=telegram discord slack ws"`
}

func (s *Service) HandleMessage(msg *Message) error {
    if err := validator.Struct(msg); err != nil {
        return fmt.Errorf("invalid message: %w", err)
    }
    // processar...
}
```

### 4. WebSocket Security

```go
// internal/infra/security/middleware.go
func WebSocketAuth(next Handler) Handler {
    return func(ctx context.Context, conn *websocket.Conn) error {
        token := conn.Header().Get("Authorization")
        if token == "" {
            return ErrUnauthorized
        }

        claims, err := validateJWT(token)
        if err != nil {
            return fmt.Errorf("%w: %v", ErrUnauthorized, err)
        }

        ctx = context.WithValue(ctx, "user", claims.UserID)
        return next(ctx, conn)
    }
}
```

### 5. Logging Seguro

```go
// RUIM: Logs com dados sensíveis
log.Info("User login", "password", user.Password)

// BOM: Não logar dados sensíveis
log.Info("User login", "user_id", user.ID)

// BOM: Maskar dados sensíveis em logs
log.Info("Request received",
    "api_key", maskString(apiKey, 4),
)
```

---

## Regras de Testes

### 1. Estratégia de Testes

```
Pirâmide de Testes:
- E2E: 1
- Integração: 5
- Unitários: 10
```

### 2. Nomenclatura

```go
// BOM: Testes com nomes descritivos
func TestOpenAIProvider_Chat_WithValidRequest_ReturnsResponse(t *testing.T)
func TestMessageRouter_Route_FromTelegramToGroq_SelectsCorrectProvider(t *testing.T)

// RUIM: Nomes genéricos
func TestProvider(t *testing.T)
func TestRoute(t *testing.T)
```

### 3. Estrutura de Teste

```go
func TestService_Chat(t *testing.T) {
    // Arrange
    mockProvider := new(MockAIProvider)
    mockProvider.On("Chat", mock.Anything, mock.Anything).Return(&ChatResponse{
        Content: "Hello!",
    }, nil)

    service := NewService(mockProvider)

    // Act
    resp, err := service.Chat(context.Background(), "Hi")

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "Hello!", resp.Content)
    mockProvider.AssertExpectations(t)
}
```

---

## Feature Development Rules

> **REGRAS OBRIGATÓRIAS para qualquer feature:**

1. **Refatoração como parte da feature**
   - Quando adicionar nova feature, refatore onde faz sentido ao invés de duplicar lógica
   - Abstração de comportamento compartilhado quando há múltiplos pontos de uso

2. **Componentes pequenos**
   - Preferir componentes pequenos e compositivos sobre caminhos grandes específicos

3. **Testes obrigatórios**
   - Toda adição de feature deve incluir testes unitários para o novo comportamento
   - Tratar resistência à regressão como parte do trabalho: não implemente nova capacidade sem cobertura de teste que proteja o caminho existente

```go
// BOM: Feature com abstração e testes
// ao invés de copiar o mesmo código em handlers diferentes:

// internal/ports/outbound/ai_provider.go
type AIProvider interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req *ChatRequest, onChunk func(string)) error
}

// feature: novo provider precisa implementar a interface
// que já existe - sem duplicação de código
```

---

## Regras de Internacionalização (i18n)

### 1. Arquitetura i18n

```
internal/infra/i18n/
├── loader.go           # Carregador deLocale
├── translator.go      # Tradutor principal
└── locales/
    ├── en.toml        # English
    ├── pt.toml        # Português
    ├── es.toml        # Español
    └── zh.toml       # Chinese
```

### 2. Uso no Código

```go
// internal/infra/i18n/translator.go
type Translator struct {
    locales map[string]map[string]string
    locale  string
}

func (t *Translator) Get(key string, args...interface{}) string {
    value, ok := t.locales[t.locale][key]
    if !ok {
        value = t.locales["en"][key]
    }

    if len(args) > 0 {
        return fmt.Sprintf(value, args...)
    }
    return value
}

// Uso com parâmetros
response := s.translator.Get("user.message.count", len(messages))
// Output: "Você tem 5 mensagens"
```

### 3. Locale Detection

```go
// Detectar Locale
func detectLocale(r *http.Request) string {
    // 1. Query parameter
    if locale := r.URL.Query().Get("locale"); locale != "" {
        return locale
    }

    // 2. Accept-Language header
    header := r.Header.Get("Accept-Language")
    locale, _ := language.MatchStrings(language.All, header)
    return locale
}
```

---

## Comandos de Desenvolvimento

### makefile

```makefile
.PHONY: build run test test-coverage docker-build docker-run docker-dev clean lint fmt tidy install-dev

# Build
build:
	go build -o bin/devilfishd ./cmd/devilfishd

# Run local
run:
	go run ./cmd/devilfishd

# Test
test:
	go test -v ./...

# Test com coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker build
docker-build:
	docker build -t devilfishd:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 -p 8081:8081 \
		-e OPENAI_API_KEY=$$OPENAI_API_KEY \
		devilfishd:latest

# Docker compose dev (hot-reload)
docker-dev:
	docker-compose -f configs/docker-compose.yaml up --build

# Clean
clean:
	rm -rf bin/ tmp/ coverage.out coverage.html

# Lint
lint:
	golangci-lint run

# Format
fmt:
	go fmt ./...
	gofmt -s -w .

# Tidy
tidy:
	go mod tidy

# Install dev tools
install-dev:
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/cmd/gosec@latest
```

---

## Docker e Hot-Reload

### Dockerfile

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /devilfishd ./cmd/devilfishd

FROM scratch

COPY --from=builder /devilfishd /devilfishd

ENTRYPOINT ["/devilfishd"]
```

### Dockerfile.dev (hot-reload)

```dockerfile
# Dockerfile.dev
FROM golang:1.22-alpine

RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["air", "-c", ".air.toml"]
```

### docker-compose.yaml

```yaml
version: '3.8'

services:
  devilfishd:
    build:
      context: .
      dockerfile: Dockerfile.dev
    volumes:
      - .:/app
      - /app/tmp
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - DEBUG=true
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - DISCORD_BOT_TOKEN=${DISCORD_BOT_TOKEN}
    env_file:
      - .env
```

### .air.toml

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
args_bin = ["run", "devilfishd"]
bin = "./tmp/main"
cmd = "go build -o ./tmp/main ./cmd/devilfishd"
delay = 1000
exclude_dir = ["assets", "tmp", "vendor", "testdata"]
exclude_file = ["*.toml"]
include_ext = ["go", "yaml", "toml", "yml"]
log = "build.log"
poll = false
rerun = false
watch = ["."]

[color]
app = ""
build = "yellow"
build_error = "red"
main = "cyan"
runner = "green"
watcher = "magenta"

[misc]
clean_on_reload = true
log_time = false

[screen]
clear_on_reload = true
keep_scroll = true
```

---

## Fluxo de Desenvolvimento

```
┌─────────────────────────────────────────────────────────────────┐
│                    FLUXO DE DESENVOLVIMENTO                       │
│                                                                 │
│  1. ANALISAR                                                     │
│     └── Ler requisitos, user stories, critérios de aceitação    │
│                                                                 │
│  2. PROJETAR                                                     │
│     ├── Definir arquitetura (ports, interfaces)                │
│     ├── Definir entidades e_VALUEObjects                        │
│     └── Definir casos de uso                                    │
│                                                                 │
│  3. IMPLEMENTAR                                                  │
│     ├── Criar entidades e value objects                         │
│     ├── Criar interfaces (ports)                               │
│     ├── Implementar adaptadores                                │
│     ├── Implementar casos de uso                                │
│     └── Configurar injeção de dependências                     │
│                                                                 │
│  4. TESTAR                                                       │
│     ├── Testes unitários (casos de uso, adaptadores)            │
│     ├── Testes de integração (se necessário)                   │
│     └── Cobertura > 80%                                        │
│                                                                 │
│  5. COMPLETAR                                                    │
│     ├── Documentar código                                      │
│     ├── Atualizar AGENTS.md se necessário                      │
│     └── Verificar contra SPEC.md                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Padrões de Projeto

### 1. Hexagonal Architecture (Ports & Adapters)

- **Ports (Inbound):** Interfaces que o mundo externo pode chamar
- **Ports (Outbound):** Interfaces que o domínio usa para comunicar com serviços externos
- **Adapters:** Implementações concretas das interfaces

### 2. Dependency Injection

- Todas as dependências são injetadas via construtores
- Facilita testabilidade e flexibilidade

### 3. Factory Pattern

- Criação de adaptadores via factories configuráveis
- Permite extensão sem modificação de código existente

### 4. Strategy Pattern

- Estratégias de roteamento de mensagens configuráveis
- Permite adicionar novos providers sem alterar código existente

### 5. Middleware Pattern

- middlewares encadeáveis para processamento de mensagens
- Permite adicionar funcionalidades como logging, keamanan, rate limiting

### 6. Repository Pattern

- Abstração de persistência de sessões
- Permite mudar de armazenamento sem alterar lógica de negocio

---

## Validação

Antes de considerar uma tarefa completa:

- [ ] Código compila sem erros
- [ ] Testes unitários passam
- [ ] Testes de integração passam (se aplicável)
- [ ] Coverage > 80%
- [ ] Lint passa
- [ ] Código segue regras de estilo
- [ ] Documentação atualizada
- [ ] AGENTS.md atualizado (se necessário)

---

## Skills System

As **Skills** formam a camada de abstração entre o Agent (LLM) e os MCP servers.

### Estrutura

```
internal/application/skill/
├── types.go           # Skill interface, ToolCall, ToolResult
├── registry.go        # Skill registry (registro e busca)
├── gmail_skills.go   # Gmail skills (send_email, list_emails, read_email)
├── drive_skills.go   # Drive skills (list_files, read_file)
└── gmail_skills_test.go
```

### Skill Interface

```go
// Skill é a interface que todas as skills devem implementar
type Skill interface {
    Name() string
    Description() string
    Schema() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
}
```

### Skills Disponíveis

| Skill | Descrição | MCP Server | Capability |
|-------|-----------|------------|------------|
| `send_email` | Envia um email via Gmail | gmail | EmailCapability |
| `list_emails` | Lista emails no Gmail | gmail | EmailCapability |
| `read_email` | Lê um email específico | gmail | EmailCapability |
| `list_files` | Lista arquivos no Google Drive | google-drive | DriveCapability |
| `read_file` | Lê um arquivo do Google Drive | google-drive | DriveCapability |
| `get_file_info` | Obtém metadados de arquivo | google-drive | DriveCapability |

### Regras para Skills

1. **Resolução via Registry**: Skills devem resolver o MCP via `MCPRegistry`
2. **Assert de Capability**: Fazer type assertion para `EmailCapability` ou `DriveCapability`
3. **Validação de Input**: Validar todos os parâmetros obrigatórios
4. **Tratamento de Erro**: Retornar erros com `fmt.Errorf` e `%w`
5. **Output Estruturado**: Sempre retornar string formatada e legível
6. **Schemas JSON**: Todo skill deve ter seu `Schema()` para o LLM

---

## Agent Loop (LLM-Driven)

O **Agent** é o motor de decisão que usa LLM para fazer escolhas e executar ferramentas.

### Estrutura

```
internal/application/agent/
├── types.go           # Step, Message, Config
├── agent.go          # Agent loop com steps, policies
├── system_prompt.go  # System prompt, tool schemas
└── agent_test.go
```

### Step Tracking

```go
// Step representa um passo na execução do agent
type Step struct {
    Thought  string                 // Razão do LLM
    ToolCall *ToolCall              // Ferramenta chamada (se houver)
    Result   string                 // Resultado da execução
    Error    error                  // Erro (se houver)
    Duration time.Duration          // Tempo de execução
}
```

### Agent Loop Flow

```
1. Adicionar user message ao contexto
2. Loop até max iterations:
   a. Chamar LLM com messages
   b. Parsear resposta para tool calls
   c. Para cada tool call:
      - Aplicar ExecutionPolicy (policies.Allow())
      - Executar skill
      - Registrar Step
      - Adicionar resultado como Message{Role: "tool"}
   d. Se sem tool calls, retornar resposta
3. Retornar resposta final ou erro
```

### System Prompt

O system prompt é injetado com:
- Regras de comportamento
- Schemas das ferramentas disponíveis
- Exemplos de formato de resposta (JSON)

---

## Execution Policy Layer (Safety Layer)

As **Policies** previnem ações inseguras antes da execução.

### Estrutura

```
internal/application/policy/
├── types.go              # ExecutionPolicy interface, PolicySet
├── email_policy.go      # Email safety policies
├── rate_limit_policy.go  # Rate limiting
├── allowed_tools_policy.go # Whitelist/blacklist
└── policy_test.go
```

### ExecutionPolicy Interface

```go
// ExecutionPolicy valida se uma tool call é permitida
type ExecutionPolicy interface {
    Allow(call ToolCall) error  // nil = permitido, error = negado
}

// PolicySet é uma coleção de policies
type PolicySet struct {
    policies []ExecutionPolicy
}
```

### Policies Disponíveis

| Policy | Descrição | Configuração |
|---------|-----------|--------------|
| `EmailPolicy` | Requer confirmação para emails, valida destinatários | `RequireConfirmation`, `AllowedDomains`, `BlockedRecipients` |
| `RateLimitPolicy` | Limita chamadas por janela de tempo | `MaxCalls`, `Window` |
| `AllowedToolsPolicy` | Whitelist de ferramentas | `[]string{"send_email", "list_files"}` |
| `BlockedToolsPolicy` | Blacklist de ferramentas | `[]string{"delete_file"}` |

### Uso

```go
// Criar policy set
ps := policy.NewPolicySet()

// Adicionar policies
ps.Add(policy.NewEmailPolicy())  // Requer confirmação para emails
ps.Add(policy.NewRateLimitPolicy(10, time.Minute))  // Max 10/min
ps.Add(policy.NewAllowedToolsPolicy([]string{"list_files", "read_file"}))

// Verificar se tool call é permitida
call := skill.ToolCall{
    Name: "send_email",
    Arguments: map[string]interface{}{
        "to": "user@example.com",
        "subject": "Hello",
        "body": "World",
        "confirmed": true,  // Necessário para EmailPolicy
    },
}
err := ps.Allow(call)
if err != nil {
    // Tool call negada
}
```

---

## Capability Interfaces

Interfaces que definem as capacidades dos MCP servers.

### EmailCapability

```go
// internal/ports/outbound/email_capability.go
type EmailCapability interface {
    SendEmail(ctx context.Context, to, subject, body string) error
    ListEmails(ctx context.Context, query string) ([]Email, error)
    ReadEmail(ctx context.Context, id string) (Email, error)
}
```

### DriveCapability

```go
// internal/ports/outbound/drive_capability.go
type DriveCapability interface {
    ListFiles(ctx context.Context, query string) ([]File, error)
    ReadFile(ctx context.Context, fileID string) (string, error)
}
```

### Implementações

- `internal/adapters/mcp/gmail/adapter.go` - GmailMCP implementa EmailCapability
- `internal/adapters/mcp/drive/adapter.go` - DriveMCP implementa DriveCapability

---

## Observabilidade

### Step Tracking

O agent registra todos os passos da execução:
- Thought (razão do LLM)
- Tool call (ferramenta usada)
- Result (resultado)
- Error (erro se houver)
- Duration (tempo de execução)

### Logging

```go
// Logs estruturados em cada etapa
logger.With(map[string]interface{}{
    "tool_name": tc.Name,
    "iteration": iteration + 1,
    "duration": time.Since(stepStart).String(),
}).Info("tool executed successfully")
```

### Acesso aos Steps

```go
// Obter steps para observabilidade
steps := agent.GetSteps()
for i, step := range steps {
    fmt.Printf("Step %d: %s\n", i+1, step.Thought)
}
```

---

## Exemplo Completo

Ver `examples/agent_with_gmail_drive/main.go` para um exemplo completo e executável que demonstra:
1. Setup de MCP registry com Gmail e Drive
2. Registro de skills
3. Configuração de policies (segurança)
4. Execução do agent loop
5. Observabilidade (steps, timing)

Para executar:
```bash
go run ./examples/agent_with_gmail_drive/
```

---

## Regras de Documentação

### Documentação Obrigatória

Toda feature implementada DEVE incluir atualização de documentação:

| Artefato | O que atualizar |
|----------|----------------|
| **README.md** | Funcionalidades, configuração, exemplos, variáveis de ambiente |
| **AGENTS.md** | Estrutura de diretórios, padrões, componentes |
| **SPEC file** | Status → "Implemented", data de implementação |

### Checklist de Documentação

Após implementação, verificar:

- [ ] README.md atualizado com a nova feature
- [ ] Seção "Funcionalidades" lista a nueva capability
- [ ] Configuração documentada (se aplicável)
- [ ] Exemplos de uso (se aplicável)
- [ ] Variáveis de ambiente listadas
- [ ] AGENTS.md atualizado (nova entidade, porta, adaptador)
- [ ] SPEC.md com status "Implemented"

### O que NÃO documentar

- Detalhes de implementação (arquivos internos)
- Decisões técnicas que pertencem ao SPEC/plan
- Código de testes (eles se auto-documentam)

### Padrão de Commits de Doc

```
docs: <descrição curta>
```

Exemplos:
- `docs: add MCP server section to README`
- `docs: update README with filesystem config`
- `docs: add MCP to AGENTS.md structure`

---

## Próximos Passos

Para iniciar o desenvolvimento:

1. Ler SPEC.md para entender arquitetura
2. Ler este AGENTS.md para regras de desenvolvimento
3. Criar.go mod e dependências básicas
4. Configurar estrutura de diretórios
5. Implementar entidades e interfaces
6. Implementar adaptadores
7. Implementar casos de uso
8. Configurar injeção de dependências
9. Configurar Docker e hot-reload

---

## Contato e Suporte

Em caso de dúvidas sobre implementação:
1. Consultar SPEC.md
2. Consultar AGENTS.md
3. Verificar testes existentes para padrões
4. Perguntar aos agentes especializados

---

## Histórico de Alterações

| Data | Versão | Alteração |
|------|--------|------------|
| 2026-04-20 | 1.0.0 | Versão inicial |
| 2026-05-02 | 1.1.0 | Gmail & Google Drive MCP integration |
| 2026-05-02 | 1.1.0 | Skills System (send_email, list_emails, read_email, list_files, read_file) |
| 2026-05-02 | 1.1.0 | Agent Safety Layer (ExecutionPolicy, EmailPolicy, RateLimitPolicy) |
| 2026-05-02 | 1.1.0 | Enhanced Agent Loop (steps, max iterations, observability) |
| 2026-05-02 | 1.1.0 | Capability interfaces (EmailCapability, DriveCapability) |
| 2026-05-02 | 1.1.0 | Minimal working example (examples/agent_with_gmail_drive/) |
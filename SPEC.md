# DevilFish - Especificação do Projeto

## Visão Geral do Projeto

**Nome:** DevilFish
**Tipo:** Harness de mensageria que conecta canais de mensagens a modelos de AI via gateway WebSocket
**Linguagem:** Go 1.22+
**Docker:** Sim

### Objetivo

Criar um harness estilo OpenClaw em Golang que:
1. Conecta múltiplos canais de mensagens (Telegram, Discord, Slack)
2. Conecta múltiplos provedores de AI (OpenAI, Groq, Gemini, Ollama)
3. Expõe um gateway WebSocket para integração externa
4. Fornece arquitetura modular e extensível

---

## Arquitetura do Sistema

### Visão de Camadas (Clean Architecture)

```
┌─────────────────────────────────────────────────────────────────┐
│                        ADAPTERS (External)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │  Telegram   │  │  Discord   │  │   Slack    │  │  WS    │ │
│  │   Adapter   │  │  Adapter   │  │  Adapter   │  │Gateway │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └───┬───┘ │
└─────────┼────────────────┼────────────────┼────────────┼──────┘
          │                │                │            │
          ▼                ▼                ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        PORTS (Interfaces)                       │
│  ┌──────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │   MessagePort    │  │   AIProvider   │  │  WebSocket    │   │
│  │    Interface     │  │   Interface    │  │   Interface   │   │
│  └────────┬─────────┘  └───────┬───────┘  └───────┬───────┘   │
└───────────┼─────────────────────┼─────────────────┼───────────┘
            │                     │                  │
            ▼                     ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DOMAIN (Core Business)                    │
│  ┌──────────────────┐  ┌────────────────┐  ┌──────────────┐  │
│  │   Message       │  │    Session     │  │   Router    │  │
│  │    Entity       │  │    Entity      │  │   Logic     │  │
│  └──────────────────┘  └────────────────┘  └──────────────┘  │
└───────────────────────────────────────���─────────────────────────┘
            │                     │                  │
            ▼                     ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     APPLICATION (Use Cases)                   │
│  ┌──────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │   HandleMessage  │  │   ChatWithAI   │  │ManageSession  │   │
│  │     UseCase      │  │    UseCase     │  │   UseCase     │   │
│  └──────────────────┘  └────────────────┘  └───────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Estrutura de Diretórios

```
devilfish/
├── cmd/
│   ├── devilfishd/              # Ponto de entrada principal
│   │   └── main.go
│   └── cli/                     # Ferramenta CLI opcional
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/              # Entidades do domínio
│   │   │   ├── message.go
│   │   │   ├── session.go
│   │   │   ├── user.go
│   │   │   └── config.go
│   │   └── valueobject/          # Value Objects
│   │       ├── messagecontent.go
│   │       └── provider.go
│   ├── application/
│   │   ├── usecase/             # Casos de uso
│   │   │   ├── handle_message.go
│   │   │   ├── chat_with_ai.go
│   │   │   ├── manage_session.go
│   │   │   └── route_message.go
│   │   └── dto/                 # Data Transfer Objects
│   │       ├── message_dto.go
│   │       └── ai_response_dto.go
│   ├── ports/
│   │   ├── inbound/            # Interfaces de entrada
│   │   │   ├── message_handler.go
│   │   │   └── websocket.go
│   │   └── outbound/            # Interfaces de saída
│   │       ├── ai_provider.go
│   │       ├── message_source.go
│   │       └── session_store.go
│   ├── adapters/
│   │   ├── messaging/           # Adaptadores de mensageria
│   │   │   ├── telegram/
│   │   │   │   ├── adapter.go
│   │   │   │   └── handler.go
│   │   │   ├── discord/
│   │   │   │   ├── adapter.go
│   │   │   │   └── handler.go
│   │   │   └── slack/
│   │   │       ├── adapter.go
│   │   │       └── handler.go
│   │   ├── ai/                  # Adaptadores de AI
│   │   │   ├── openai/
│   │   │   │   └── provider.go
│   │   │   ├── groq/
│   │   │   │   └── provider.go
│   │   │   ├── gemini/
│   │   │   │   └── provider.go
│   │   │   └── ollama/
│   │   │       └── provider.go
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
���   │   │   ├── loader.go
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
├── Makefile
└── README.md
```

---

## Padrões de Projeto Utilizados

### 1. Hexagonal Architecture (Ports & Adapters)

**Implementação:**
- **Ports (Inbound):** Interfaces que o mundo externo pode chamar (MessageHandler, WebSocket)
- **Ports (Outbound):** Interfaces que o domínio usa para comunicar com serviços externos (AIProvider, MessageSource)
- **Adapters:** Implementações concretas das interfaces

**Exemplo:**
```go
// internal/ports/outbound/ai_provider.go
type AIProvider interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req *ChatRequest, onChunk func(string)) error
}

// internal/adapters/ai/openai/provider.go
type OpenAIProvider struct {
    client *openai.Client
    model  string
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // implementação concreta
}
```

### 2. Dependency Injection via Constructor

**Regra:** Todas as dependências são injetadas via construtores
**Benefício:** Testabilidade, flexibilidade, baixo acoplamento

```go
func NewHandleMessageUseCase(
    aiProvider ports.AIProvider,
    messageHandler ports.MessageHandler,
    sessionStore ports.SessionStore,
) *HandleMessageUseCase {
    return &HandleMessageUseCase{
        aiProvider:     aiProvider,
        messageHandler: messageHandler,
        sessionStore:   sessionStore,
    }
}
```

### 3. Factory Pattern para Adaptadores

**Regra:** Criação de adaptadores via factories configuráveis

```go
// internal/adapters/adapter_factory.go
type AdapterFactory struct {
    config *config.Config
}

func (f *AdapterFactory) CreateAIProvider(name string) (ports.AIProvider, error) {
    switch name {
    case "openai":
        return NewOpenAIProvider(f.config.OpenAI)
    case "groq":
        return NewGroqProvider(f.config.Groq)
    // ...
    }
    return nil, fmt.Errorf("unknown provider: %s", name)
}
```

### 4. Strategy Pattern para Routamento

**Regra:** Estratégias de roteamento de mensagens configuráveis

```go
type RouterStrategy interface {
    Route(ctx context.Context, msg *entity.Message) (*RouteDecision, error)
}

type AIChatStrategy struct {
    providers map[string]ports.AIProvider
    sessionStore ports.SessionStore
}
```

### 5. Middleware Pattern

**Regra:** middlewares encadeáveis para processamento de mensagens

```go
type MessageMiddleware func(next MessageHandler) MessageHandler

func Chain(middlewares ...MessageMiddleware) MessageMiddleware {
    return func(next MessageHandler) MessageHandler {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}
```

### 6. Repository Pattern para Sessões

```go
type SessionRepository interface {
    Get(ctx context.Context, id string) (*entity.Session, error)
    Save(ctx context.Context, session *entity.Session) error
    Delete(ctx context.Context, id string) error
    ListByUser(ctx context.Context, userID string) ([]*entity.Session, error)
}
```

---

## Regras de Desenvolvimento

### 1. Código

#### 1.1 Estrutura e Organização

| Regra | Descrição |
|-------|-----------|
| **Pacotes small** | Máx 15 arquivos por pacote |
| **Nomes significativos** | Variáveis/expressões autoexplicativas |
| **Funções small** | Máx 50 linhas por função |
| **Structs small** | Máx 8 campos por struct |
| **DRY** | Evitar duplicação, usar abstractions |
| **Cohesion** |related código junto) |

#### 1.2 Padrões de Código Go

```go
// ✓ BOM: Interfaces pequenas e focadas
type MessageHandler interface {
    Handle(ctx context.Context, msg *entity.Message) error
}

// ✗ RUIM: Interface grande
type EverythingHandler interface {
    Handle(Message)
    Process(AI)
    Save(Session)
    Connect(WS)
    // ...
}

// ✓ BOM: Construtor com dependências claras
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

#### 1.3 tratamento de Erros

```go
// ✓ BOM: Erros wrapados com contexto
if err != nil {
    return fmt.Errorf("failed to send message to %s: %w", channelID, err)
}

// ✓ BOM: Errors customizados
var ErrProviderNotAvailable = errors.New("AI provider not available")

func (s *Service) Chat(ctx context.Context, msg string) error {
    if !s.provider.IsAvailable() {
        return fmt.Errorf("%w: %s", ErrProviderNotAvailable, s.provider.Name())
    }
    // ...
}
```

#### 1.4 Logging

```go
// ✓ BOM: Structured logging
s.logger.Info("handling message",
    "message_id", msg.ID,
    "user_id", msg.UserID,
    "channel", msg.Channel,
)

// ✗ RUIM: Strings concatenadas
s.logger.Info("handling message: " + msg.ID)
```

### 2. Refatoração

#### 2.1 Quando Refatorar

| Sinal | Ação |
|-------|------|
| 3+ duplicações | Extrair função auxiliar |
| Função > 50 linhas | Quebrar em funções menores |
| Struct > 8 campos | Quebrar em structs menores |
| Parâmetros > 4 | Usar struct para parâmetros |
| switch grande | Usar strategy/map |
| Many if-else chain | Usar polymorphism |

#### 2.2 Extraindo Código

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

### 3. Testes

#### 3.1 Estratégia de Testes

```
┌─────────────────────────────────────────────┐
│           Pirâmide de Testes                 │
│                                             │
│                ▲                           │
│               /│\        E2E (1)            │
│              / │ \                         │
│             /  │  \    Integração (5)     │
│            /───│───\                        │
│           /    │    \   Unitários (10)      │
│          ╱───���─���─────\                       │
│         ───────────────────                    │
└─────────────────────────────────────────────┘
```

#### 3.2 Nomenclatura

```go
// ✓ BOM: Testes com nomes descritivos
func TestOpenAIProvider_Chat_WithValidRequest_ReturnsResponse(t *testing.T)
func TestMessageRouter_Route_FromTelegramToGroq_SelectsCorrectProvider(t *testing.T)

// ✗ RUIM: Nomes genéricos
func TestProvider(t *testing.T)
func TestRoute(t *testing.T)
```

#### 3.3 Struktur de Teste

```go
func TestService_Chat(t *testing.T) {
    // Arrange
    mockProvider := new.MockAIProvider()
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

### 4. Concorrência

#### 4.1 Goroutines

```go
// ✓ BOM: goroutine com contexto
go func() {
    defer wg.Done()
    err := processWithContext(ctx, item)
    resultChan <- result{item, err}
}()

// ✓ BOM: usando Worker Pool
func worker(id int, jobs <-chan Job, results chan<- Result) {
    for job := range jobs {
        results <- process(job)
    }
}
```

#### 4.2 Channels

```go
// ✓ BOM: channel com tamanho definido
jobs := make(chan Job, 100)  // avoid blocking

// ✓ BOM: usando context para cancelamento
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-resultChan:
    return result
}

// ✗ RUIM: channel não fechado
// ✗ RUIM: leak de goroutine
```

#### 4.3 Mutex vs Channels

| Situação | Recomendação |
|---------|-------------|
| Estado compartilhado simples | sync.Mutex |
| Passing ownership | Channel |
| Signaling de eventos | Channel |
| Contagem de recursos | sync.WaitGroup |
| Uma vez setup | sync.Once |

---

## Regras de Segurança

### 1. Configuração Sensível

#### 1.1 Variáveis de Ambiente

```yaml
# .env.example (NUNCA commitar)
# configs/config.yaml
ai:
  openai:
    api_key: ${OPENAI_API_KEY}  # ✓ BOM
    # api_key: "sk-xxx"       # ✗ RUIM: hardcoded

messaging:
  telegram:
    bot_token: ${TELEGRAM_BOT_TOKEN}
  discord:
    bot_token: ${DISCORD_BOT_TOKEN}
```

#### 1.2 Secrets no Docker

```yaml
# docker-compose.yaml
services:
  devilfishd:
    secrets:
      - openai_api_key
      - telegram_token

secrets:
  openai_api_key:
    external: true
  telegram_token:
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
// ✓ BOM: Validar toda entrada
type Message struct {
    Content string `json:"content" validate:"required,min=1,max=4000"`
    UserID  string `json:"user_id" validate:"required,uuid4"`
    Channel string `json:"channel" validate:"required,oneof=telegram discord slack ws"`
}

func (s *Service) handleMessage(msg *Message) error {
    if err := validator.Struct(msg); err != nil {
        return fmt.Errorf("invalid message: %w", err)
    }
    // processar...
}
```

### 4. CSRF e XSS

```go
// Sanitizar entrada do usuário
func sanitizeInput(input string) string {
    // Remover HTML tags
    re := regexp.MustCompile(`<[^>]*>`)
    sanitized := re.ReplaceAllString(input, "")

    // Escapar caracteres especiais
    return html.EscapeString(sanitized)
}
```

### 5. WebSocket Security

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

### 6. Logging Seguro

```go
// ✗ RUIM: Logs com dados sensíveis
log.Info("User login", "password", user.Password)

// ✓ BOM: Não logar dados sensíveis
log.Info("User login", "user_id", user.ID)

// ✓ BOM: Maskar dados sensíveis em logs
log.Info("Request received",
    "api_key", maskString(apiKey, 4),
)
```

---

## Internacionalização (i18n)

### 1. Arquitetura i18n

```
internal/infra/i18n/
├── loader.go           # Carregador deLocale
├── translator.go     # Tradutor principal
└── locales/
    ├── en.toml       # English
    ├── pt.toml       # Português
    ├── es.toml       # Español
    └── zh.toml       # Chinese
```

### 2. Locale Structure

```toml
# locales/pt.toml
[ai]
response.error = "Erro ao processar sua mensagem. Tente novamente."
response.timeout = "Tempo esgotado. Por favor, tente novamente."
response.rate_limit = "Muitas mensagens. Aguarde um momento."

[telegram]
welcome = "Bem-vindo ao DevilFish! Como posso ajudar?"
help = "Use /help para ver os comandos disponíveis"

[discord]
welcome = "Hello! I'm DevilFish. How can I help?"
help = "Use /help para ver os comandos disponíveis"

[slack]
mention = "Hey! Como posso ajudar?"
```

### 3. Uso no Código

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

// Uso
func (s *Service) handleMessage(ctx context.Context, msg *entity.Message) error {
    // ...
    response := s.translator.Get("ai.response.error")

    // Com parâmetros
    response := s.translator.Get("user.message.count", len(messages))
    // Output: "Você tem 5 mensagens"
}
```

### 4. Locale detection

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

### 5. Configuração

```yaml
# configs/config.yaml
i18n:
  default_locale: "en"
  supported_locales:
    - "en"
    - "pt"
    - "es"
    - "zh"
  fallback_locale: "en"
```

---

## Hot-Reload com Air

### 1. Configuração Air

```toml
# .air.toml
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
poll_interval = 0
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

### 2. Docker com Hot-Reload

```yaml
# docker-compose.dev.yaml
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
    command: air -c .air.toml

# Dockerfile.dev
FROM golang:1.22-alpine AS builder

RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["air"]
```

---

## Comandos Úteis

### Makefile

```makefile
# Makefile
.PHONY: build run test test-coverage docker-build docker-run clean lint fmt

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

# Docker compose dev
docker-dev:
	docker-compose -f docker-compose.dev.yaml up --build

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

## Fluxo de Dados

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         FLUXO DE MENSAGEM                             │
│                                                                      │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│  │ Mensagem │───▶│ Valida   │───▶│  Routing │───▶│AI Provider│     │
│  │ Recebida │    │ entrada  │    │Decisão   │    │ Chat     │     │
│  └──────────┘    └──────────┘    └──────────┘    └────┬─────┘     │
│                                                         │           │
│                                                         ▼           │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    │
│  │  Session │◀───│ Armazena │◀───│ Response │◀───│Format   │    │
│  │  Update  │    │  Estado  │    │  Formata │    │Resposta │    │
│  └──────────┘    └──────────┘    └──────────┘    └─────────┘    │
│                                                         │           │
│                                                         ▼           │
│                                              ┌──────────────────┐ │
│                                              │ Retorna p/ Canal │ │
│                                              │ Original         │ │
│                                              └──────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

---

## Configuração Completa

### Exemplo config.yaml

```yaml
# configs/config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  ws_port: 8081
  read_timeout: 30s
  write_timeout: 30s

i18n:
  default_locale: "en"
  supported_locales:
    - "en"
    - "pt"
    - "es"
  fallback_locale: "en"

ai:
  providers:
    - name: "openai"
      model: "gpt-4-turbo-preview"
      api_key: ${OPENAI_API_KEY}
      base_url: "https://api.openai.com/v1"
      enabled: true

    - name: "groq"
      model: "llama-3-70b"
      api_key: ${GROQ_API_KEY}
      base_url: "https://api.groq.com/openai/v1"
      enabled: true

    - name: "gemini"
      model: "gemini-pro"
      api_key: ${GEMINI_API_KEY}
      base_url: "https://generativelanguage.googleapis.com/v1"
      enabled: false

    - name: "ollama"
      model: "llama3"
      base_url: "http://localhost:11434"
      enabled: true

  default_provider: "openai"
  timeout: 60s
  max_retries: 3

messaging:
  telegram:
    enabled: true
    bot_token: ${TELEGRAM_BOT_TOKEN}
    webhook_url: ${TELEGRAM_WEBHOOK_URL}
    rate_limit:
      requests: 30
      window: 60s

  discord:
    enabled: true
    bot_token: ${DISCORD_BOT_TOKEN}
    intents: ["GUILDS", "GUILD_MESSAGES"]
    rate_limit:
      requests: 10
      window: 60s

  slack:
    enabled: true
    bot_token: ${SLACK_BOT_TOKEN}
    signing_secret: ${SLACK_SIGNING_SECRET}
    rate_limit:
      requests: 10
      window: 60s

websocket:
  server:
    enabled: true
    host: "0.0.0.0"
    port: 8081
  auth:
    enabled: true
    jwt_secret: ${JWT_SECRET}
  rate_limit:
    requests: 100
    window: 60s

routing:
  rules:
    - pattern: ".*"
      provider: "openai"
      system_prompt: "You are DevilFish, a helpful AI assistant."

session:
  storage:
    type: "memory"  # ou "redis"
    redis:
      addr: ${REDIS_ADDR}
      password: ${REDIS_PASSWORD}
      db: 0
  ttl: 24h

logging:
  level: "info"
  format: "json"
  output: "stdout"

security:
  rate_limit:
    enabled: true
    default:
      requests: 50
      window: 60s
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
```

---

## API endpoints

### WebSocket Gateway

```
WS ws://host:port/ws?token=<jwt_token>&locale=en

// Mensagem de entrada
{
  "type": "chat",
  "content": "Hello!",
  "session_id": "optional-session-id",
  "provider": "optional-provider-name"
}

// Mensagem de saída
{
  "type": "chat",
  "content": "Hello! How can I help?",
  "session_id": "session-id",
  "model": "gpt-4-turbo-preview",
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

### REST API (Opcional)

```
GET  /health
POST /api/v1/chat
POST /api/v1/session
GET  /api/v1/session/:id
DELETE /api/v1/session/:id
```

---

## Métricas e Observabilidade

### Métricas (Prometheus)

```go
var (
    MessagesProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "devilfish_messages_total",
            Help: "Total number of messages processed",
        },
        []string{"channel", "status"},
    )

    AIRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "devilfish_ai_request_duration_seconds",
            Help:    "Duration of AI requests in seconds",
            Buckets: []float64{.1, .5, 1, 5, 10},
        },
        []string{"provider", "model"},
    )
)
```

---

## Conclusão

Este documento define a arquitetura completa para o projeto DevilFish:
- Clean Architecture com Ports & Adapters
- Padrões de projeto Go estabelecidos
- Regras de código, refatoração e testes
- Segurança completa
- Internacionalização
- Hot-reload configurado
- Docker-ready

Próximos passos: Implementar o código base seguindo esta especificação.
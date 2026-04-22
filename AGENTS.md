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
│   │   │   ├── message.go
│   │   │   ├── session.go
│   │   │   ├── user.go
│   │   │   └── config.go
│   │   └── valueobject/       # Value Objects
│   │       ├── messagecontent.go
│   │       └── provider.go
│   ├── application/
│   │   ├── usecase/          # Casos de uso
│   │   │   ├── handle_message.go
│   │   │   ├── chat_with_ai.go
│   │   │   ├── manage_session.go
│   │   │   └── route_message.go
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
│   │       └── session_store.go
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
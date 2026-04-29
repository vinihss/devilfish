# DevilFish

Harness de mensageria que conecta canais de mensagens a modelos de AI via gateway WebSocket.

## Visão Geral

DevilFish é um servidor em Go que atua como ponte entre múltiplos canais de mensagens (Telegram, Discord, Slack) e provedores de AI (OpenAI, Groq, Gemini, Ollama), exposto via gateway WebSocket para integração externa.

### Funcionalidades

- **Multi-canal**: Telegram, Discord, Slack
- **Multi-provider**: OpenAI, Groq, Gemini, Ollama
- **MCP Servers**: Filesystem, GitHub, Memory (extensible)
- **Gateway WebSocket**: Integração em tempo real
- **Arquitetura modular**: Hexagonal Architecture (Ports & Adapters)
- **Docker-ready**: Pronto para produção com multi-stage build
- **Hot-reload**: Desenvolvimento local com Air

## Arquitetura

DevilFish é composto por dois runtimes independentes que se comunicam via HTTP:

```
┌──────────────────────────────────┐      HTTP POST       ┌─────────────────────────────────────┐
│       messagingd                 │  /api/message ──────► │           devilfishd (core)         │
│  (messaging runtime)             │ ◄────────────         │  (AI + WebSocket + MCP)             │
│                                  │                       │                                     │
│  ┌─────────┐ ┌─────────┐        │                       │  ┌──────────┐  ┌────────────────┐  │
│  │Telegram │ │Discord  │        │                       │  │ ChatWithAI│  │ WebSocket GW  │  │
│  └────┬────┘ └────┬────┘        │                       │  └────┬─────┘  └───────┬────────┘  │
│  ┌─────────┐      │             │                       │       │               │             │
│  │ Slack   │      │             │                       │  ┌────┴──────────────┴────────┐    │
│  └────┬────┘      │             │                       │  │   inbound.MessageHandler   │    │
│       │           │             │                       │  └────────────────────────────┘    │
│  ┌────▼───────────▼────────┐   │                       │                                     │
│  │  GatewayClient          │   │                       │  ┌──────────────┐ ┌──────────────┐ │
│  │ (MessageHandler via HTTP)│  │                       │  │  OpenAI/Groq │ │  MCP Servers │ │
│  └─────────────────────────┘  │                       │  └──────────────┘ └──────────────┘ │
└──────────────────────────────────┘                       └─────────────────────────────────────┘
```

### Runtimes

| Runtime | Binário | Porta padrão | Responsabilidade |
|---------|---------|-------------|-----------------|
| Core | `devilfishd` | 8082 (HTTP) / 8083 (WS) | IA, sessões, WebSocket, endpoint `/api/message` |
| Messaging | `messagingd` | 8084 (HTTP) | Webhooks Telegram/Discord/Slack |

Os adaptadores de mensageria nunca têm dependência direta de provedores de IA — eles só conhecem a interface `inbound.MessageHandler`, implementada pelo `GatewayClient`.

### Arquitetura interna (Hexagonal)

```
┌─────────────────────────────────────────────────────────┐
│                      ADAPTERS                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │Telegram │  │Discord  │  │ Slack   │  │   WS    │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  │
└───────┼────────────┼────────────┼────────────┼─────────┘
        │            │            │             │
        ▼            ▼            ▼             ▼
┌─────────────────────────────────────────────────────────┐
│                       PORTS                             │
│  ┌──────────────┐  ┌────────────┐  ┌────────────────┐  │
│  │ MessagePort  │  │AIProvider  │  │  WebSocket     │  │
│  └─────┬────────┘  └────┬──────┘  └───────┬────────┘  │
│  ┌─────────────┐                                        │
│  │  MCPClient  │                                        │
│  └─────┬───────┘                                        │
└─────────────────────────────────────────────────────────┘
        │                │                  │
        ▼                ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                   APPLICATION                           │
│  ┌────────────────┐  ┌────────────┐  ┌───────────────┐ │
│  │HandleMessage   │  │ ChatWithAI │  │ ManageSession │ │
│  └────────────────┘  └────────────┘  └───────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### Pré-requisitos

- Go 1.22+
- Docker (opcional)
- Tokens de API dos provedores

### 1. Clone o repositório

```bash
git clone https://github.com/user/devilfish.git
cd devilfish
```

### 2. Configure variáveis de ambiente

```bash
cp configs/config.example.yaml configs/config.yaml
# Edite config.yaml com suas credenciais
```

### 3. Execute os dois runtimes

```bash
# Terminal 1 — Core runtime (IA + WebSocket)
make run

# Terminal 2 — Messaging runtime (Telegram/Discord/Slack)
make run-messaging
```

### 4. Ou com Docker

```bash
docker compose -f configs/docker-compose.yaml up --build
```

## Configuração

### Estrutura do config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8082      # Core HTTP port (also serves /api/message)
  ws_port: 8083   # WebSocket port

# Gateway: messaging runtime uses this to reach the core
gateway:
  url: "http://localhost:8082"
  api_key: ${GATEWAY_API_KEY:-}  # Optional shared secret

ai:
  providers:
    - name: "openai"
      model: "gpt-4o"
      api_key: ${OPENAI_API_KEY}
      enabled: true
    - name: "groq"
      model: "llama-3.1-8b-instant"
      api_key: ${GROQ_API_KEY}
      enabled: true

messaging:
  server:
    host: "0.0.0.0"
    port: 8084    # Messaging runtime HTTP port
  telegram:
    bot_token: ${TELEGRAM_BOT_TOKEN}
    enabled: true
  discord:
    bot_token: ${DISCORD_BOT_TOKEN}
    enabled: true

websocket:
  auth:
    enabled: true
    jwt_secret: ${JWT_SECRET}
```

### Variáveis de ambiente

| Variável | Runtime | Descrição |
|----------|---------|-----------|
| `OPENAI_API_KEY` | core | Chave da API da OpenAI |
| `GROQ_API_KEY` | core | Chave da API do Groq |
| `JWT_SECRET` | core | Segredo para JWT (WebSocket auth) |
| `GATEWAY_API_KEY` | ambos | Shared secret entre os runtimes (opcional) |
| `TELEGRAM_BOT_TOKEN` | messaging | Token do bot do Telegram |
| `DISCORD_BOT_TOKEN` | messaging | Token do bot do Discord |
| `SLACK_BOT_TOKEN` | messaging | Token do bot do Slack |

## MCP Server Connection

O DevilFish suporta conexão com servidores MCP (Model Context Protocol) para estender suas capacidades com ferramentas externas.

### Servidores Suportados

| Servidor | Transport | Ferramentas |
|----------|----------|------------|
| `filesystem` | stdio | read_file, write_file, list_directory, create_directory, etc. |
| `github` | http | get_file, create_issue, search_repositories, etc. |
| `memory` | stdio | memory_read, memory_write, memory_delete, memory_search |

### Configuração

Adicione servidores MCP na seção `mcp.servers` do config.yaml:

```yaml
mcp:
  servers:
    - name: "filesystem"
      enabled: true
      transport: "stdio"
      command: "npx"
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/tmp"
      timeout: 30
      max_retries: 3

    - name: "github"
      enabled: true
      transport: "http"
      url: "http://localhost:3000"
      auth_token: ${MCP_GITHUB_TOKEN}
      timeout: 30
      max_retries: 3

    - name: "memory"
      enabled: true
      transport: "stdio"
      command: "npx"
      args:
        - "-y"
        - "@modelcontextprotocol/server-memory"
      timeout: 30
      max_retries: 3
```

### Instalação de Pré-requisitos

Para servidores stdio, instale os pacotes Node.js:

```bash
# Filesystem server
npm install -g @modelcontextprotocol/server-filesystem

# Memory server
npm install -g @modelcontextprotocol/server-memory
```

### Variáveis de ambiente

| Variável | Descrição |
|----------|----------|
| `MCP_GITHUB_TOKEN` | Token de acesso GitHub (para servidor github) |

### Uso Programático

```go
import (
    "devilfish/internal/adapters/mcp"
    "devilfish/internal/ports/outbound"
)

// Criar pool de conexões
pool := mcp.NewPool()

// Adicionar servidor
config := outbound.MCPServerConfig{
    Name:      "filesystem",
    Transport: "stdio",
    Command:  "npx",
    Args:     []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
}
client, err := pool.Add(context.Background(), config)

// Listar ferramentas
tools, err := client.ListTools(context.Background())

// Executar ferramenta
result, err := client.ExecuteTool(context.Background(), "read_file", map[string]interface{}{
    "path": "/tmp file.txt",
})
```

## API

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8081/ws?token=<jwt_token>');

// Enviar mensagem
ws.send(JSON.stringify({
  type: 'chat',
  content: 'Olá!',
  session_id: 'user-123',
  provider: 'openai'  // opcional
}));

// Receber resposta
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.content);
};
```

### REST

```
GET  /health            # Core runtime health check
POST /api/message       # Internal: messaging runtime → core runtime
GET  /ws                # WebSocket gateway

# Messaging runtime
GET  /health                 # Messaging runtime health check
POST /webhooks/telegram      # Telegram webhook
POST /webhooks/discord       # Discord webhook
POST /webhooks/slack         # Slack webhook
```

## Desenvolvimento

### Comandos do Makefile

```bash
make build           # Build core runtime (devilfishd)
make build-messaging # Build messaging runtime (messagingd)
make build-all       # Build both binaries
make run             # Run core runtime locally
make run-messaging   # Run messaging runtime locally
make test            # Rodar testes
make lint            # Verificar lint
make docker-dev      # Docker com hot-reload
```

### Hot-Reload com Air

```bash
make docker-dev
```

Isso inicia o container com hot-reload via Air. Mudanças em arquivos Go são recompiladas automaticamente.

### Estrutura de diretórios

```
devilfish/
├── cmd/
│   ├── devilfishd/            # Core runtime (IA + WebSocket + /api/message)
│   ├── messagingd/            # Messaging runtime (Telegram/Discord/Slack)
│   └── cli/                   # CLI management tool
├── internal/
│   ├── domain/
│   │   ├── entity/            # Message, Session, User, MCPServer, Tool
│   │   └── valueobject/       # Provider, MessageContent, MCPConnection
│   ├── application/           # Casos de uso
│   ├── ports/
│   │   ├── inbound/           # Interfaces de entrada (MessageHandler)
│   │   └── outbound/          # AIProvider, MCPClient, SessionStore
│   ├── adapters/
│   │   ├── ai/                # OpenAI, Groq, Gemini, Ollama
│   │   ├── messaging/         # Telegram, Discord, Slack
│   │   ├── gatewayclient/     # HTTP client → /api/message (used by messagingd)
│   │   ├── mcp/               # MCP client, transports, pool, registry
│   │   ├── websocket/         # WebSocket gateway
│   │   └── storage/           # Session store
│   └── infra/                 # Config, logging, i18n, security
├── pkg/                       # Pacotes reutilizáveis
├── configs/                   # Arquivos de configuração
└── tests/                     # Testes unitários
```

## Internacionalização

O sistema suporta múltiplos idiomas:

- `en` - English
- `pt` - Português
- `es` - Español
- `zh` - Chinese

Arquivos de locale em `internal/infra/i18n/locales/`.

## Observabilidade

### Logs estruturados

O sistema usa logs estruturados em formato JSON:

```go
logger.Info("handling message",
    "message_id", msg.ID,
    "user_id", msg.UserID,
    "channel", msg.Channel,
)
```

### Métricas Prometheus

- `devilfish_messages_total` - Total de mensagens processadas
- `devilfish_ai_request_duration_seconds` - Duração das requisições à AI

## Segurança

- **Secrets**: Via variáveis de ambiente, nunca hardcoded
- **Rate limiting**: Token bucket por usuário/canal
- **JWT**: Autenticação obrigatória no WebSocket
- **Input validation**: Validação de toda entrada externa

## License

MIT
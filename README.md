# DevilFish

Harness de mensageria que conecta canais de mensagens a modelos de AI via gateway WebSocket.

## Visão Geral

DevilFish é um servidor em Go que atua como ponte entre múltiplos canais de mensagens (Telegram, Discord, Slack) e provedores de AI (OpenAI, Groq, Gemini, Ollama), exposto via gateway WebSocket para integração externa.

### Funcionalidades

- **Multi-canal**: Telegram, Discord, Slack
- **Multi-provider**: OpenAI, Groq, Gemini, Ollama
- **MCP Servers**: Filesystem, GitHub, Memory, **Gmail**, **Google Drive** (extensible)
- **Skills System**: Granular skills for tool execution (send_email, list_files, read_file, etc.)
- **Agent Safety Layer**: Execution policies (email confirmation, rate limiting, tool whitelist/blacklist)
- **Agent Loop**: LLM-driven decision engine with step tracking and observability
- **Gateway WebSocket**: Integração em tempo real
- **Arquitetura modular**: Hexagonal Architecture (Ports & Adapters)
- **Docker-ready**: Pronto para produção com multi-stage build
- **Hot-reload**: Desenvolvimento local com Air

## Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│                      ADAPTERS                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │Telegram │  │Discord  │  │ Slack   │  │   WS    │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  │
└───────┼────────────┼────────────┼────────────┼─────────┘
        │           │           │            │
        ▼           ▼           ▼            ▼
┌─────────────────────────────────────────────────────────┐
│                       PORTS                            │
│  ┌──────────────┐  ┌────────────┐  ┌────────────────┐  │
│  │ MessagePort │  │AIProvider │  │  WebSocket     │  │
│  └─────┬──────┘  └────┬─────┘  └───────┬────────┘  │
│  ┌─────────────┐                                            │
│  │ MCPClient  │                                            │
│  └─────┬─────┘                                            │
└───────┼───────────────┼──────────────────┼──────────┘
        │              │                  │
        ▼              ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                      DOMAIN                            │
│  ┌──────────────┐  ┌────────────┐  ┌──────────────┐  │
│  │  Message   │  │ Session  │  │  Router   │  │
│  └──────────────┘  └──────────┘  └───────────┘  │
│  ┌──────────────┐  ┌────────────┐                    │
│  │  MCPServer │  │  Tool    │                    │
│  └──────────────┘  └────────────┘                    │
└─────────────────────────────────────────────────────────┘
        │              │                  │
        ▼              ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                   APPLICATION                          │
│  ┌────────────────┐  ┌────────────┐  ┌───────────┐  │
│  │HandleMessage   │  │ ChatWithAI│  │ManageSess│  │
│  └────────────────┘  └─��────────┘  └──────────┘  │
└─────────────────────────────────────────────────────────┘
```
┌─────────────────────────────────────────────────────────┐
│                      ADAPTERS                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │Telegram │  │Discord  │  │ Slack   │  │   WS    │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  │
└───────┼────────────┼────────────┼────────────┼─────────┘
        │           │           │            │
        ▼           ▼           ▼            ▼
┌─────────────────────────────────────────────────────────┐
│                       PORTS                            │
│  ┌──────────────┐  ┌────────────┐  ┌────────────────┐  │
│  │ MessagePort │  │AIProvider│  │  WebSocket     │  │
│  └─────┬──────┘  └────┬─────┘  └───────┬────────┘  │
└───────┼───────────────┼──────────────────┼──────────┘
        │              │                  │
        ▼              ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                      DOMAIN                            │
│  ┌──────────────┐  ┌────────────┐  ┌──────────────┐  │
│  │  Message   │  │ Session  │  │  Router   │  │
│  └────────────┘  └──────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────┘
        │              │                  │
        ▼              ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                   APPLICATION                          │
│  ┌────────────────┐  ┌────────────┐  ┌───────────┐  │
│  │HandleMessage   │  │ ChatWithAI│  │ManageSess│  │
│  └────────────────┘  └──────────┘  └──────────┘  │
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

### 3. Execute localmente

```bash
make run
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
  port: 8080
  ws_port: 8081

ai:
  providers:
    - name: "openai"
      model: "gpt-4-turbo-preview"
      api_key: ${OPENAI_API_KEY}
      enabled: true
    - name: "groq"
      model: "llama-3-70b"
      api_key: ${GROQ_API_KEY}
      enabled: true
  system_prompt: "file://prompts/agents/default_system_prompt.txt"

messaging:
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

`ai.system_prompt` aceita texto puro **ou** referência para arquivo:

- Texto puro: `system_prompt: "Você é um assistente..."`
- Arquivo: `system_prompt: "file://prompts/agents/default_system_prompt.txt"`

### Variáveis de ambiente obrigatórias

| Variável | Descrição |
|----------|----------|
| `OPENAI_API_KEY` | Chave da API da OpenAI |
| `GROQ_API_KEY` | Chave da API do Groq |
| `TELEGRAM_BOT_TOKEN` | Token do bot do Telegram |
| `DISCORD_BOT_TOKEN` | Token do bot do Discord |
| `JWT_SECRET` | Segredo para JWT (WebSocket auth) |

## MCP Server Connection

O DevilFish suporta conexão com servidores MCP (Model Context Protocol) para estender suas capacidades com ferramentas externas.

### Servidores Suportados

| Servidor | Transport | Ferramentas |
|----------|----------|------------|
| `filesystem` | stdio | read_file, write_file, list_directory, create_directory, etc. |
| `github` | http | get_file, create_issue, search_repositories, etc. |
| `memory` | stdio | memory_read, memory_write, memory_delete, memory_search |
| `gmail` | stdio/http | send_email, list_emails, read_email |
| `google-drive` | stdio/http | list_files, read_file |

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

    - name: "gmail"
      enabled: true
      transport: "stdio"
      command: "npx"
      args:
        - "-y"
        - "@modelcontextprotocol/server-gmail"
      timeout: 30
      max_retries: 3

    - name: "google-drive"
      enabled: true
      transport: "stdio"
      command: "npx"
      args:
        - "-y"
        - "@modelcontextprotocol/server-google-drive"
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

## Skills System

O DevilFish possui um sistema de **Skills** que atua como camada de abstração entre o Agent (LLM) e os MCP servers.

### Skills Disponíveis

| Skill | Descrição | MCP Server |
|------|-----------|------------|
| `send_email` | Envia um email via Gmail | gmail |
| `list_emails` | Lista emails no Gmail | gmail |
| `read_email` | Lê um email específico | gmail |
| `list_files` | Lista arquivos no Google Drive | google-drive |
| `read_file` | Lê um arquivo do Google Drive | google-drive |
| `get_file_info` | Obtém metadados de um arquivo | google-drive |

### Uso Programático

```go
import (
    "devilfish/internal/application/skill"
    "devilfish/internal/adapters/mcp"
)

// Criar registro de skills
registry := skill.NewRegistry()

// Registrar skills (precisa do MCP registry)
registry.Register(skill.NewSendEmailSkill(mcpRegistry))
registry.Register(skill.NewListEmailsSkill(mcpRegistry))
registry.Register(skill.NewReadEmailSkill(mcpRegistry))
registry.Register(skill.NewListFilesSkill(mcpRegistry))
registry.Register(skill.NewReadFileSkill(mcpRegistry))

// Executar skill
result, err := registry.ExecuteSkill(ctx, "list_files", map[string]interface{}{
    "query": "name contains 'report'",
})
```

## Agent Safety Layer

O DevilFish inclui uma camada de **Execution Policies** para prevenir ações inseguras.

### Políticas Disponíveis

| Política | Descrição |
|-----------|-----------|
| `EmailPolicy` | Requer confirmação para envio de emails, valida destinatários |
| `RateLimitPolicy` | Limita número de chamadas de ferramentas por janela de tempo |
| `AllowedToolsPolicy` | Whitelist de ferramentas permitidas |
| `BlockedToolsPolicy` | Blacklist de ferramentas bloqueadas |

### Uso Programático

```go
import (
    "devilfish/internal/application/agent"
    "devilfish/internal/application/policy"
)

// Criar conjunto de políticas
policies := policy.NewPolicySet()

// Adicionar políticas
policies.Add(policy.NewEmailPolicy()) // Requer confirmação para emails
policies.Add(policy.NewRateLimitPolicy(10, time.Minute)) // Max 10 chamadas/min
policies.Add(policy.NewAllowedToolsPolicy([]string{"list_files", "read_file"})) // Apenas estas ferramentas

// Criar agente com políticas
agent := agent.NewAgent(
    agent.Config{
        MaxIterations: 10,
        Model: "gpt-4",
    },
    skillRegistry,
    policies,
    aiProvider,
    logger,
)

// Executar
response, err := agent.Run(ctx, "Send the file report.pdf to john@example.com")
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

### REST (opcional)

```
GET  /health
POST /api/v1/chat
POST /api/v1/session
GET  /api/v1/session/:id
DELETE /api/v1/session/:id
```

## Desenvolvimento

### Comandos do Makefile

```bash
make build      # Build binário
make run       # Executar localmente
make test      # Rodar testes
make lint      # Verificar lint
make docker    # Build e run Docker
make docker-dev # Docker com hot-reload
```

### Hot-Reload com Air

```bash
make docker-dev
```

Isso inicia o container com hot-reload via Air. Mudanças em arquivos Go são recompiladas automaticamente.

### Estrutura de diretórios

```
devilfish/
├── cmd/                    # Entry points
├── internal/
│   ├── domain/
│   │   ├── entity/        # Message, Session, User, MCPServer, Tool
│   │   └── valueobject/   # Provider, MessageContent, MCPConnection
│   ├── application/       # Casos de uso
│   ├── ports/
│   │   ├── inbound/      # Interfaces de entrada
│   │   └── outbound/    # AIProvider, MCPClient, SessionStore
│   ├── adapters/
│   │   ├── ai/         # OpenAI, Groq, Gemini, Ollama
│   │   ├── messaging/   # Telegram, Discord, Slack
│   │   ├── mcp/        # MCP client, transports, pool, registry
│   │   ├── websocket/  # WebSocket gateway
│   │   └── storage/   # Session store
│   └── infra/          # Config, logging, i18n, security
├── pkg/                  # Pacotes reutilizáveis
├── configs/              # Arquivos de configuração
└── tests/               # Testes unitários
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

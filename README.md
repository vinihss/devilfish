# DevilFish

Harness de mensageria que conecta canais de mensagens a modelos de AI via gateway WebSocket.

## Visão Geral

DevilFish é um servidor em Go que atua como ponte entre múltiplos canais de mensagens (Telegram, Discord, Slack) e provedores de AI (OpenAI, Groq, Gemini, Ollama), exposto via gateway WebSocket para integração externa.

### Funcionalidades

- **Multi-canal**: Telegram, Discord, Slack
- **Multi-provider**: OpenAI, Groq, Gemini, Ollama
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

### Variáveis de ambiente obrigatórias

| Variável | Descrição |
|----------|----------|
| `OPENAI_API_KEY` | Chave da API da OpenAI |
| `GROQ_API_KEY` | Chave da API do Groq |
| `TELEGRAM_BOT_TOKEN` | Token do bot do Telegram |
| `DISCORD_BOT_TOKEN` | Token do bot do Discord |
| `JWT_SECRET` | Segredo para JWT (WebSocket auth) |

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
│   ├── domain/            # Entidades e value objects
│   ├── application/      # Casos de uso
│   ├── ports/           # Interfaces
│   ├── adapters/        # Implementações concretas
│   └── infra/          # Config, logging, i18n
├── pkg/                  # Pacotes reutilizáveis
├── configs/              # Arquivos de configuração
└── tests/               # Testes
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
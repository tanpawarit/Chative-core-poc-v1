# Chative Core POC v1

A proof-of-concept conversational agent in Go that composes an agent graph with CloudWeGo Eino, uses Google Gemini models for NLU and response generation, and persists conversation history in Redis. The entry point (`main.go`) demonstrates an end-to-end flow configured via environment variables.

![Agent Flow](@asset/agent_flow.png)

## Features
- Agent graph with conditional branches (NLU → parse → human handoff or response assembly → tools → final response)
- Dual-model setup: NLU model and Response model (Gemini)
- Tool calling workflow with input sanitization and call limits
- Conversation history in Redis with TTL per conversation
- Structured error handling (`internal/core/error`) and Zerolog-based logging (`pkg/logger`)
- Prompt rendering via Eino prompt components with templates

## Project Structure
```
internal/
  agent/
    graph/
      conversations/   # Conversation context assembly
      nodes/           # Eino nodes + state handlers
      observers/       # Prompt/model/tool callbacks
      parsers/         # NLU parser
      prompts/         # Prompt renderers + templates
      tools/           # Tool definitions and registry
    model/             # Agent data models and configs
    repo/              # Conversation repository impls (Redis)
  core/
    environment.go     # Environment helpers
    error/             # Unified error type + wrappers
pkg/
  logger/              # Zerolog wrapper (+autoload)
  redis/               # Redis client config
main.go                # Demo building and invoking the graph
```

Key files:
- Graph composition: `internal/agent/graph/graph.go`
- Nodes and state: `internal/agent/graph/nodes/nodes.go`
- Chat models (Gemini): `internal/agent/graph/nodes/chat_models.go`
- NLU parser: `internal/agent/graph/parsers/nlu_parser.go`
- Prompts: `internal/agent/graph/prompts/*.go` and `internal/agent/graph/prompts/template/*`
- Tools registry: `internal/agent/graph/tools/manager.go`
- Conversation repo interface: `internal/agent/model/conversation.go`
- Redis repo: `internal/agent/repo/conversation.go`
- Errors: `internal/core/error/*.go`
- Logger: `pkg/logger/logger.go`
- Redis config: `pkg/redis/redis.go`

## Prerequisites
- Go 1.25+
- A Redis instance (local or hosted) and connection URL
- Google Gemini API key

## Quickstart
1) Copy environment example and fill secrets
```
cp .env.example .env
# Edit .env and set at least:
#   GEMINI_API_KEY=...
#   REDIS_URL=redis://... (or rediss:// for TLS)
```

2) Run the demo
```
go run .
```
This builds the agent graph, connects to Redis, and runs sample queries from `main.go`.

## Configuration
Environment variables (see `.env.example`):
- Core
  - `ENVIRONMENT` = development|staging|testing|production
- Gemini
  - `GEMINI_API_KEY` (required)
  - `GEMINI_BASE_URL` (optional override)
- Redis
  - `REDIS_URL`
  - `REDIS_READ_TIMEOUT`, `REDIS_WRITE_TIMEOUT`, `REDIS_DIAL_TIMEOUT`
- NLU model
  - `NLU_MODEL`, `NLU_MAX_TOKENS`, `NLU_TEMPERATURE`
  - `NLU_DEFAULT_INTENT`, `NLU_ADDITIONAL_INTENT`
  - `NLU_DEFAULT_ENTITY`, `NLU_ADDITIONAL_ENTITY`
- Response model
  - `RESPONSE_MODEL`, `RESPONSE_MAX_TOKENS`, `RESPONSE_TEMPERATURE`
- Prompt
  - `PROMPT_BUSINESS_TYPE`, `PROMPT_BUSINESS_NAME`
- Conversation/session
  - `CONVERSATION_TTL`, `CONVERSATION_NLU_MAX_TURNS`, `CONVERSATION_TOOL_MAX_CALLS`

`main.go` loads `.env` using `github.com/joho/godotenv` and binds to a typed config using `github.com/kelseyhightower/envconfig`.

## How It Works
1) InputConverter: Saves the user message and prepares NLU context from recent turns.
2) NLUChatModel: Runs NLU model (Gemini) on the context.
3) Parser: Converts NLU model output into `NLUResponse` with safety limits.
4) Branch A (negative sentiment): Human handoff message.
5) Branch B: ResponseAssembler creates system prompt using NLU analysis and builds conversation context.
6) ResponseChatModel: Generates assistant response; may emit tool calls.
7) ToolExecutor: Executes registered tools sequentially with sanitization and a call limit; loops back to the response model.
8) Finalization: Saves the assistant’s final content message into Redis.

Cost tracking: Node post-handlers compute per-call model usage cost and accumulate it in the per-request state.

## Extending the Agent
- Add a tool: Implement under `internal/agent/graph/tools/` and register in `GetQueryTools()`.
- Tune prompts: Edit templates under `internal/agent/graph/prompts/template/` and adjust renderers.
- Change models: Update env vars in `.env` (model name, temperature, max tokens).
- Persistence: Swap/extend the repository via `internal/agent/model.ConversationRepository`.

## Troubleshooting
- Redis connection: Validate `REDIS_URL` and network reachability. Redis errors are normalized by `WrapRedis`.
- Missing/invalid env: Ensure required keys exist; `envconfig` reports missing values on startup.
- Prompt or parser errors: Check logs; the parser emits structured error hints in `ParsingMetadata`.

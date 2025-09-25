# Chative Core POC v1 — Go Development Guideline

## Project Overview

This repository is a proof‑of‑concept conversational agent built in Go. It composes an agent graph using CloudWeGo Eino, calls Google Gemini chat models for NLU and response generation, and persists conversation history in Redis. The entry point (`main.go`) demonstrates an end‑to‑end flow driven entirely by environment configuration.

## Project Structure

```
├── .env                 # Local environment (do not commit secrets)
├── .env.example         # Example env for local setup
├── go.mod               # Module: github.com/Chative-core-poc-v1/server
├── internal/
│   ├── agent/
│   │   ├── graph/
│   │   │   ├── conversations/      # Conversation context assembly
│   │   │   │   └── manager.go
│   │   │   ├── nodes/               # Eino nodes and state handlers
│   │   │   ├── observers/           # Prompt/model/tool callbacks
│   │   │   ├── parsers/             # NLU parser
│   │   │   ├── prompts/             # Prompt renderers + templates
│   │   │   └── tools/               # Tool definitions and registry
│   │   ├── model/                   # Agent data models and configs
│   │   └── repo/                    # Conversation repository impls
│   └── core/
│       ├── environment.go           # Env type helpers
│       └── error/                   # Unified error type + wrappers
├── pkg/
│   ├── logger/                      # Zerolog wrapper
│   │   └── autoload/                # Optional auto-init (import for side-effects)
│   └── redis/                       # Redis client config
└── main.go                          # Demo driver building and invoking the graph
```

## Architecture

- Agent Graph: Built with Eino Compose. High‑level flow:
  - InputConverter → NLUChatModel → Parser → (HumanHandoff | ResponseAssembler) → ResponseChatModel → (ToolExecutor | END)
  - Branching is controlled by conditions (negative sentiment → HumanHandoff, tool calls present → ToolExecutor).
- Conversation History: `internal/agent/model.ConversationRepository` interface with a Redis implementation at `internal/agent/repo/conversation.go`.
- Prompts: Rendered via Eino prompt components from templates in `internal/agent/graph/prompts/template/`.
- Tools: Business tools are defined under `internal/agent/graph/tools/` and are bound to the response model.
- Errors: Unified `errx.Error` with helpers (see `internal/core/error`). Redis errors are normalized via `WrapRedis`.
- Logging: Lightweight wrapper around Zerolog in `pkg/logger`.

## Key Components

- Conversation Repository: `internal/agent/model/conversation.go`
  - Contract for `AddMessage`, `LoadHistory`, `ClearHistory`, `GetMessageCount`.
  - Redis implementation: `internal/agent/repo/conversation.go` uses a per‑conversation Redis list with TTL.

- Messages Manager: `internal/agent/graph/conversations/manager.go`
  - Prepares NLU context from recent messages and builds response context with the system prompt.

- Graph Builder: `internal/agent/graph/graph.go`
  - Creates chat models, binds tools, sets up nodes/edges/branches, compiles runnable graph, and returns a `Runner` with `Invoke`.

- Nodes and State: `internal/agent/graph/nodes/`
  - Pre/post handlers manage per‑invocation `AppState` (conversation ID, history, tool call counters, accumulated usage cost).

- Prompts: `internal/agent/graph/prompts/`
  - `RenderNLUSystem` and `RenderResponseSystem` render system prompts and trigger Eino prompt callbacks.

- Tools: `internal/agent/graph/tools/`
  - Registry in `manager.go` exposes tools to the response model; unknown tool calls are handled gracefully.

## Configuration

Environment variables (see `.env.example`):
- `ENVIRONMENT` development|staging|testing|production
- `GEMINI_API_KEY` required
- `GEMINI_BASE_URL` optional override (empty uses default)
- `REDIS_URL` connection string; optional timeouts:
  - `REDIS_READ_TIMEOUT`, `REDIS_WRITE_TIMEOUT`, `REDIS_DIAL_TIMEOUT`
- NLU model: `NLU_MODEL`, `NLU_MAX_TOKENS`, `NLU_TEMPERATURE`, `NLU_DEFAULT_INTENT`, `NLU_ADDITIONAL_INTENT`, `NLU_DEFAULT_ENTITY`, `NLU_ADDITIONAL_ENTITY`
- Response model: `RESPONSE_MODEL`, `RESPONSE_MAX_TOKENS`, `RESPONSE_TEMPERATURE`
- Prompt: `PROMPT_BUSINESS_TYPE`, `PROMPT_BUSINESS_NAME`
- Conversation: `CONVERSATION_TTL`, `CONVERSATION_NLU_MAX_TURNS`, `CONVERSATION_TOOL_MAX_CALLS`

Config loading happens in `main.go` via `github.com/kelseyhightower/envconfig` after loading `.env` with `github.com/joho/godotenv`.

## Running Locally

- Copy `.env.example` to `.env` and fill required secrets.
- Ensure Redis is reachable (local or hosted). The app reads `REDIS_URL`.
- Run the demo:
  - `go run .` to build the agent graph and execute the sample queries in `main.go`.
  - Adjust test queries and `conversationID` in `main.go` as needed.

## Extending the Agent

- Add a Tool
  - Implement under `internal/agent/graph/tools/`.
  - Register in `internal/agent/graph/tools/manager.go` via `GetQueryTools()`.
  - The response model is bound to tool infos during graph setup.

- Modify Prompts
  - Edit templates in `internal/agent/graph/prompts/template/`.
  - Rendering logic is in `internal/agent/graph/prompts/*.go`.

- Change Models/Behavior
  - Tweak env variables in `.env` (models, temperatures, token limits, tool call cap, NLU intents/entities).

- Persistence Layer
  - The repository interface is in `internal/agent/model/conversation.go`.
  - The default Redis implementation is in `internal/agent/repo/conversation.go` and uses JSON‑encoded Eino messages.

## Error Handling & Logging

- Use `internal/core/error` (`errx`) for structured errors.
  - `WrapRedis` maps Redis errors to HTTP‑ish status codes and public messages.
- Use `pkg/logger` for logs. To auto‑init, import `pkg/logger/autoload` for side‑effects, or call `logx.Init()` explicitly.
- LLM usage cost is computed in node post‑handlers and accumulated per request in `AppState.TotalCostUSD`.

## Code Standards

- Packages/files: short, lowercase; types and exported funcs: PascalCase; unexported: camelCase.
- Keep business logic in nodes/managers; repositories are persistence only.
- Validate inputs at the edges (env/config, tool args sanitization in `nodes` pre‑handlers).
- Table‑driven tests recommended for parsers and utilities.

## Troubleshooting

- Redis issues: verify `REDIS_URL` and network reachability; `redis.Nil` is normalized by `WrapRedis`.
- Missing API key: ensure `GEMINI_API_KEY` is set; `envconfig` will fail the process if required vars are missing.
- Prompt/JSON parsing: see `internal/agent/graph/parsers/nlu_parser.go` for safety limits and error annotations.

## References

- CloudWeGo Eino: https://github.com/cloudwego/eino
- Zerolog: https://github.com/rs/zerolog
- go-redis: https://github.com/redis/go-redis
- envconfig: https://github.com/kelseyhightower/envconfig

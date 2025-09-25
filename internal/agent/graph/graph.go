package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	logx "github.com/Chative-core-poc-v1/server/pkg/logger"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/Chative-core-poc-v1/server/internal/agent/graph/conversations"
	"github.com/Chative-core-poc-v1/server/internal/agent/graph/nodes"
	"github.com/Chative-core-poc-v1/server/internal/agent/graph/observers"
	"github.com/Chative-core-poc-v1/server/internal/agent/graph/tools"
	"github.com/Chative-core-poc-v1/server/internal/agent/model"
)

// Runner is a thin wrapper to execute the compiled graph with the public QueryInput.
type Runner interface {
	Invoke(ctx context.Context, in model.QueryInput) (string, error)
}

// Config holds everything needed to compose the full response graph end-to-end.
// This is a convenience layer over GraphConfig that also constructs ChatModels and MessagesManager.
type Config struct {
	APIKey           string
	BaseURL          string
	NLUModel         model.NLUModelConfig
	ResponseModel    model.ResponseModelConfig
	ResponsePrompt   model.ResponsePromptConfig
	Conversation     model.ConversationConfig
	ConversationRepo model.ConversationRepository
}

// GraphConfig holds all configuration needed to build the graph
type GraphConfig struct {
	ChatModels           *nodes.ChatModels
	MessagesManager      *conversations.MessagesManager
	NLUConfig            *model.NLUModelConfig
	ResponsePromptConfig *model.ResponsePromptConfig
	ToolMaxCalls         int
}

// GraphBuilder handles the construction of the agent conversation graph
type GraphBuilder struct {
	config *GraphConfig
	graph  *compose.Graph[model.QueryInput, *schema.Message]
}

type graphRunner struct {
	runnable compose.Runnable[model.QueryInput, *schema.Message]
}

func (r *graphRunner) Invoke(ctx context.Context, in model.QueryInput) (string, error) {
	// TODO: Add comprehensive error handling and recovery
	// - Implement circuit breaker pattern for external dependencies
	// - Add retry logic with exponential backoff for transient failures
	// - Include detailed error context and correlation IDs for debugging
	// - Add timeout handling with configurable deadlines

	out, err := r.runnable.Invoke(ctx, model.QueryInput{
		ConversationID: in.ConversationID,
		Query:          in.Query,
	}, compose.WithCallbacks(observers.NewAllCallbacks()))
	if err != nil {
		return "", err
	}
	if out == nil {
		return "", nil
	}
	// Best-effort print Extra (e.g., usage_cost) if present
	if len(out.Extra) > 0 {
		if b, err := json.MarshalIndent(out.Extra, "", "  "); err == nil {
			fmt.Printf("Extra: %s\n", string(b))
		}
	}
	return out.Content, nil
}

// BuildResponseGraph composes ChatModels, MessagesManager, builds the graph, and returns a Runner.
func BuildResponseGraph(ctx context.Context, cfg Config) (Runner, error) {
	if cfg.ConversationRepo == nil {
		return nil, fmt.Errorf("conversation repo is nil")
	}

	// Create chat models
	cms, err := nodes.NewChatModels(ctx, nodes.ChatModelConfig{
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		NLUConfig:  &cfg.NLUModel,
		RespConfig: &cfg.ResponseModel,
	})
	if err != nil {
		return nil, err
	}

	// Create messages manager
	mm := conversations.NewMessagesManager(cfg.ConversationRepo, cfg.Conversation)

	// Build runnable graph
	runnable, err := BuildGraph(ctx, &GraphConfig{
		ChatModels:           cms,
		MessagesManager:      mm,
		NLUConfig:            &cfg.NLUModel,
		ResponsePromptConfig: &cfg.ResponsePrompt,
		ToolMaxCalls:         cfg.Conversation.Tools.MaxCalls,
	})
	if err != nil {
		return nil, err
	}

	logx.Debug().Msg("Response graph built successfully")
	return &graphRunner{runnable: runnable}, nil
}

// BuildGraph constructs and returns the compiled agent graph
func BuildGraph(ctx context.Context, config *GraphConfig) (compose.Runnable[model.QueryInput, *schema.Message], error) {
	// Basic config validation
	if config == nil {
		return nil, fmt.Errorf("graph config is nil")
	}
	if config.ChatModels == nil || config.ChatModels.NLU == nil || config.ChatModels.Response == nil {
		return nil, fmt.Errorf("chat models are not properly initialized")
	}
	if config.MessagesManager == nil {
		return nil, fmt.Errorf("messages manager is nil")
	}
	if config.NLUConfig == nil || config.ResponsePromptConfig == nil {
		return nil, fmt.Errorf("model prompt/config is nil")
	}

	builder := &GraphBuilder{
		config: config,
		graph: compose.NewGraph[model.QueryInput, *schema.Message](
			compose.WithGenLocalState(func(ctx context.Context) *model.AppState {
				return &model.AppState{}
			}),
		),
	}

	if err := builder.setupTools(ctx); err != nil {
		return nil, err
	}

	builder.addNodes()
	builder.addEdges()

	if err := builder.addBranches(); err != nil {
		return nil, err
	}

	return builder.compile(ctx)
}

// setupTools configures business tools and binds them to the response model
func (b *GraphBuilder) setupTools(ctx context.Context) error {
	businessTools := tools.GetQueryTools()
	toolInfos, err := tools.GetToolInfos(ctx, businessTools)
	if err != nil {
		logx.Error().Err(err).Msg("Failed to get tool infos")
		return fmt.Errorf("failed to get tool infos: %w", err)
	}

	if err := b.config.ChatModels.BindToolsToResponseModel(ctx, toolInfos); err != nil {
		logx.Error().Err(err).Msg("Failed to bind tools to response model")
		return fmt.Errorf("failed to bind tools to response model: %w", err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools:               businessTools,
		ExecuteSequentially: true,
		UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
			// Gracefully handle hallucinated or malformed tool calls (e.g., empty name)
			logx.Warn().
				Str("tool_name", name).
				Str("arguments", input).
				Msg("Unknown or invalid tool call; returning fallback result")
			// Return a compact, structured message the model can use to proceed
			return fmt.Sprintf("{\"error\":\"unknown_tool\",\"name\":%q,\"note\":\"ignored\"}", name), nil
		},
		ToolArgumentsHandler: func(ctx context.Context, name, arguments string) (string, error) {
			// Best-effort sanitize; never fail hard here
			var m map[string]any
			if err := json.Unmarshal([]byte(arguments), &m); err != nil {
				// keep original if not JSON
				return arguments, nil
			}

			switch name {
			case tools.ToolSearchProduct:
				// query: string (required)
				if v, ok := m["query"]; ok {
					switch vv := v.(type) {
					case string:
						m["query"] = strings.TrimSpace(vv)
					default:
						// coerce non-string to string
						m["query"] = strings.TrimSpace(fmt.Sprint(v))
					}
				}
				// category: string (optional)
				if v, ok := m["category"]; ok {
					switch vv := v.(type) {
					case string:
						m["category"] = strings.TrimSpace(vv)
					default:
						delete(m, "category")
					}
				}
				// max_results: number (optional, default 10, max 20)
				if v, ok := m["max_results"]; ok {
					switch vv := v.(type) {
					case float64:
						// JSON numbers decode as float64
						m["max_results"] = clampInt(int(vv), 1, 20)
					case string:
						if n, err := strconv.Atoi(strings.TrimSpace(vv)); err == nil {
							m["max_results"] = clampInt(n, 1, 20)
						} else {
							delete(m, "max_results")
						}
					default:
						delete(m, "max_results")
					}
				}
			case tools.ToolGetProductDetails:
				// product_id: string (required)
				if v, ok := m["product_id"]; ok {
					switch vv := v.(type) {
					case string:
						m["product_id"] = strings.TrimSpace(vv)
					default:
						m["product_id"] = strings.TrimSpace(fmt.Sprint(v))
					}
				}
			}

			b, err := json.Marshal(m)
			if err != nil {
				// fallback to original
				return arguments, nil
			}
			return string(b), nil
		},
	})
	if err != nil {
		logx.Error().Err(err).Msg("Failed to create tools node")
		return fmt.Errorf("failed to create tools node: %w", err)
	}

	b.graph.AddToolsNode(nodes.NodeToolExecutor, toolsNode,
		compose.WithStatePreHandler(nodes.NewToolExecutorPreHandler(b.config.ToolMaxCalls)),
	)

	return nil
}

// addNodes adds all processing nodes to the graph
func (b *GraphBuilder) addNodes() {
	b.graph.AddLambdaNode(nodes.NodeInputConverter,
		nodes.NewInputConverterNode(b.config.MessagesManager, b.config.NLUConfig),
		compose.WithStatePreHandler(nodes.NewInputConverterPreHandler()),
	)

	b.graph.AddChatModelNode(nodes.NodeNLUChatModel,
		nodes.NewNLUChatModelNode(b.config.ChatModels.NLU),
		compose.WithStatePostHandler(nodes.NewNLUChatModelPostHandler(b.config.ChatModels.NLUModelName)),
	)

	b.graph.AddLambdaNode(nodes.NodeParser,
		nodes.NewParserNode(),
		compose.WithStatePostHandler(nodes.NewParserPostHandler()),
	)

	b.graph.AddLambdaNode(nodes.NodeResponseAssembler,
		nodes.NewResponseAssemblerNode(b.config.MessagesManager, b.config.ResponsePromptConfig),
	)

	b.graph.AddLambdaNode(nodes.NodeHumanHandoff,
		nodes.NewHumanHandoffNode(),
	)

	b.graph.AddChatModelNode(nodes.NodeResponseChatModel,
		nodes.NewResponseChatModelNode(b.config.ChatModels.Response),
		compose.WithStatePreHandler(nodes.NewResponseChatModelPreHandler(b.config.ToolMaxCalls)),
		compose.WithStatePostHandler(nodes.NewResponseChatModelPostHandler(b.config.MessagesManager, b.config.ChatModels.ResponseModelName)),
	)
}

// addEdges creates the main flow connections between nodes
func (b *GraphBuilder) addEdges() {
	edges := [][2]string{
		{compose.START, nodes.NodeInputConverter},
		{nodes.NodeInputConverter, nodes.NodeNLUChatModel},
		{nodes.NodeNLUChatModel, nodes.NodeParser},
		{nodes.NodeHumanHandoff, compose.END},
		{nodes.NodeResponseAssembler, nodes.NodeResponseChatModel},
		{nodes.NodeToolExecutor, nodes.NodeResponseChatModel},
	}

	for _, edge := range edges {
		b.graph.AddEdge(edge[0], edge[1])
	}
}

// addBranches creates conditional routing branches
func (b *GraphBuilder) addBranches() error {
	handoffBranch := compose.NewGraphBranch(
		nodes.NewHumanHandoffCondition(),
		map[string]bool{
			nodes.NodeHumanHandoff:      true,
			nodes.NodeResponseAssembler: true,
		},
	)
	if err := b.graph.AddBranch(nodes.NodeParser, handoffBranch); err != nil {
		logx.Error().Err(err).Msg("Error adding human handoff branch")
		return fmt.Errorf("error adding human handoff branch: %w", err)
	}

	decisionBranch := compose.NewGraphBranch(
		nodes.NewToolExecutorCondition(),
		map[string]bool{
			nodes.NodeToolExecutor: true,
			compose.END:            true,
		},
	)
	if err := b.graph.AddBranch(nodes.NodeResponseChatModel, decisionBranch); err != nil {
		logx.Error().Err(err).Msg("Error adding decision branch")
		return fmt.Errorf("error adding decision branch: %w", err)
	}

	return nil
}

// compile finalizes and compiles the graph
func (b *GraphBuilder) compile(ctx context.Context) (compose.Runnable[model.QueryInput, *schema.Message], error) {
	// Limit total run steps to avoid infinite loops in branching or tool retries
	maxSteps := 10 + b.config.ToolMaxCalls*2
	if maxSteps < 20 {
		maxSteps = 20
	}

	runnable, err := b.graph.Compile(ctx, compose.WithMaxRunSteps(maxSteps))
	if err != nil {
		logx.Error().Err(err).Msg("Error compiling graph")
		return nil, fmt.Errorf("error compiling graph: %w", err)
	}

	logx.Debug().Msg("Graph compiled successfully")
	return runnable, nil
}

// clampInt returns v limited to [min, max].
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

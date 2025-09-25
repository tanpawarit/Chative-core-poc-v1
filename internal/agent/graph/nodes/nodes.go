package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/Chative-core-poc-v1/server/internal/agent/graph/conversations"
	"github.com/Chative-core-poc-v1/server/internal/agent/graph/parsers"
	"github.com/Chative-core-poc-v1/server/internal/agent/graph/prompts"
	"github.com/Chative-core-poc-v1/server/internal/agent/model"
	logx "github.com/Chative-core-poc-v1/server/pkg/logger"
)

// NewInputConverterPreHandler creates the pre-handler for InputConverter node
func NewInputConverterPreHandler() func(context.Context, model.QueryInput, *model.AppState) (model.QueryInput, error) {
	return func(ctx context.Context, in model.QueryInput, s *model.AppState) (model.QueryInput, error) {
		if s.ConversationID == "" {
			s.ConversationID = in.ConversationID
		}
		// Reset tool call counter and limit flag for each new query
		s.ToolCallCount = 0
		s.ToolCallLimitReached = false
		s.ToolCallIDSeq = 0
		// Reset accumulated total cost for each new query
		s.TotalCostUSD = 0
		return in, nil
	}
}

// TODO: recheck context for all models nodes
// NewInputConverterNode creates the InputConverter node for NLU processing
func NewInputConverterNode(
	mm *conversations.MessagesManager,
	nluCfg *model.NLUModelConfig,
) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input model.QueryInput) ([]*schema.Message, error) {
		conversationCtx, err := mm.ProcessNLUMessage(ctx, input.ConversationID, input.Query)
		if err != nil {
			return nil, fmt.Errorf("error getting conversation context: %w", err)
		}

		// Generate system prompt via Eino prompt component (enables prompt callbacks)
		systemPrompt, err := prompts.RenderNLUSystem(ctx, nluCfg)
		if err != nil {
			return nil, fmt.Errorf("render nlu system prompt: %w", err)
		}

		// Create messages with customerID in Extra
		messages := []*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(conversationCtx),
		}

		return messages, nil
	})
}

// NewNLUChatModelPostHandler computes and logs usage cost for the NLU model.
func NewNLUChatModelPostHandler(modelName string) func(context.Context, *schema.Message, *model.AppState) (*schema.Message, error) {
	return func(ctx context.Context, out *schema.Message, state *model.AppState) (*schema.Message, error) {
		if model.CostEnabled() && out != nil && out.ResponseMeta != nil && out.ResponseMeta.Usage != nil {
			pricing := model.ResolvePricing(modelName)
			inC, outC, totalC := model.ComputeCost(out.ResponseMeta.Usage, pricing)
			if out.Extra == nil {
				out.Extra = map[string]any{}
			}
			out.Extra["usage_cost"] = map[string]any{
				"currency":          "USD",
				"model":             modelName,
				"prompt_tokens":     out.ResponseMeta.Usage.PromptTokens,
				"completion_tokens": out.ResponseMeta.Usage.CompletionTokens,
				"total_tokens":      out.ResponseMeta.Usage.TotalTokens,
				"input_cost":        inC,
				"output_cost":       outC,
				"total_cost":        totalC,
			}
			logx.Debug().
				Str("conversation_id", state.ConversationID).
				Str("node", NodeNLUChatModel).
				Str("model", modelName).
				Int("prompt_tokens", out.ResponseMeta.Usage.PromptTokens).
				Int("completion_tokens", out.ResponseMeta.Usage.CompletionTokens).
				Int("total_tokens", out.ResponseMeta.Usage.TotalTokens).
				Float64("input_cost_usd", inC).
				Float64("output_cost_usd", outC).
				Float64("total_cost_usd", totalC).
				Msg("LLM usage")

			// Accumulate only total cost into state
			state.TotalCostUSD += totalC

			// Also expose running total in the message Extra for visibility
			out.Extra["usage_cost_total_usd"] = state.TotalCostUSD
		}
		return out, nil
	}
}

// NewParserNode creates the Parser node for NLU response parsing
func NewParserNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, resp *schema.Message) (model.NLUResponse, error) {
		result, err := parsers.ParseNLUResponse(resp.Content)
		if err != nil {
			logx.Error().Err(err).Msg("Error parsing NLU response")
			return model.NLUResponse{}, err
		}
		if result == nil {
			logx.Error().Msg("Parsing returned nil result")
			return model.NLUResponse{}, fmt.Errorf("parsing returned nil result")
		}
		return *result, nil
	})
}

// NewParserPostHandler creates the post-handler for Parser node
func NewParserPostHandler() func(context.Context, model.NLUResponse, *model.AppState) (model.NLUResponse, error) {
	return func(ctx context.Context, out model.NLUResponse, state *model.AppState) (model.NLUResponse, error) {
		// Save NLU to State
		state.NLUAnalysis = &out

		importanceScore := out.ImportanceScore
		conversationID := state.ConversationID
		logx.Debug().
			Str("conversation_id", conversationID).
			Float64("importance_score", importanceScore).
			Msg("Evaluating importance score")

		// TODO: Implement Episodic Memory real database save
		if importanceScore > 0.7 {
			logx.Debug().
				Float64("importance_score", importanceScore).
				Msg("High importance score detected - would save to Episodic Memory database")
		} else {
			logx.Debug().
				Float64("importance_score", importanceScore).
				Msg("Importance score below threshold - skipping Episodic Memory database save")
		}
		return out, nil
	}
}

// NewHumanHandoffCondition creates the condition function for routing to human handoff
func NewHumanHandoffCondition() func(context.Context, model.NLUResponse) (string, error) {
	return func(ctx context.Context, input model.NLUResponse) (string, error) {
		s := input.Sentiment
		if s.Label == "negative" && s.Confidence > 0.94 {
			logx.Debug().Str("sentiment_label", s.Label).Float64("sentiment_confidence", s.Confidence).
				Msg("Routing to admin - high confidence negative sentiment detected")
			return NodeHumanHandoff, nil
		}
		logx.Debug().Str("sentiment_label", s.Label).Float64("sentiment_confidence", s.Confidence).
			Msg("Routing to Response Assembler - no human alert needed")
		return NodeResponseAssembler, nil
	}
}

// NewHumanHandoffNode creates the HumanHandoff node for escalating negative sentiment cases
func NewHumanHandoffNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input model.NLUResponse) (*schema.Message, error) {
		sentiment := input.Sentiment
		logx.Warn().
			Str("sentiment_label", sentiment.Label).
			Float64("sentiment_confidence", sentiment.Confidence).
			Msg("Human intervention required for negative sentiment")

		// TODO: Implement actual escalation logic (e.g., notify admin, create ticket, etc.)
		// Return a message indicating human intervention is needed
		return schema.SystemMessage("Human intervention required for negative sentiment. Case escalated to admin."), nil
	})
}

// NewResponseAssemblerNode creates the ResponseAssembler node for building response context
func NewResponseAssemblerNode(
	mm *conversations.MessagesManager,
	responsePromptConfig *model.ResponsePromptConfig,
) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, nluResult model.NLUResponse) ([]*schema.Message, error) {
		// Get data from state
		var data model.ResponseData
		err := compose.ProcessState(ctx, func(_ context.Context, state *model.AppState) error {
			if state.NLUAnalysis == nil {
				return fmt.Errorf("missing NLU analysis in state")
			}
			data = model.ResponseData{
				Analysis:       *state.NLUAnalysis,
				ConversationID: state.ConversationID,
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to access state: %w", err)
		}

		// Generate system prompt with NLU analysis via Eino prompt component (enables prompt callbacks)
		respSysPrompt, err := prompts.RenderResponseSystem(ctx, *responsePromptConfig, data.Analysis)
		if err != nil {
			return nil, fmt.Errorf("generate response prompt: %w", err)
		}

		// Build context with conversation history
		messages, err := mm.BuildResponseContext(ctx, data.ConversationID, respSysPrompt)
		if err != nil {
			return nil, fmt.Errorf("build response context: %w", err)
		}

		return messages, nil
	})
}

// NewResponseChatModelPreHandler creates the pre-handler for ResponseChatModel node
func NewResponseChatModelPreHandler(maxToolCalls int) func(context.Context, []*schema.Message, *model.AppState) ([]*schema.Message, error) {
	return func(ctx context.Context, in []*schema.Message, state *model.AppState) ([]*schema.Message, error) {
		// Heuristic fix for Gemini OpenAI-compat: ensure tool results carry tool_call_id
		if len(in) > 0 {
			last := in[len(in)-1]
			if last != nil && last.Role == schema.Tool && strings.TrimSpace(last.ToolCallID) == "" {
				// Try to find the most recent assistant tool call id from history
				for i := len(state.History) - 1; i >= 0; i-- {
					msg := state.History[i]
					if msg == nil || msg.Role != schema.Assistant || len(msg.ToolCalls) == 0 {
						continue
					}
					// Note: ToolCall struct is from Eino schema and typically contains ID/Name/Args
					id := msg.ToolCalls[0].ID
					if strings.TrimSpace(id) != "" {
						last.ToolCallID = id
					}
					break
				}
			}
		}

		state.History = append(state.History, in...)

		if checkAndMarkToolLimit(state, maxToolCalls) {
			maxToolCalls = normalizeMaxToolCalls(maxToolCalls)
			wrapUp := &schema.Message{
				Role: schema.System,
				Content: fmt.Sprintf(
					"SYSTEM NOTICE: You have reached the maximum tool call limit (%d). "+
						"Please synthesize a helpful response using the information you've already gathered. "+
						"Acknowledge any limitations in your response if you couldn't complete all necessary tool calls.",
					maxToolCalls,
				),
			}
			state.History = append(state.History, wrapUp)
		}

		logx.Debug().Msg("AI thinking...")

		return state.History, nil
	}
}

// NewResponseChatModelPostHandler creates the post-handler for ResponseChatModel node
func NewResponseChatModelPostHandler(
	mm *conversations.MessagesManager,
	modelName string,
) func(context.Context, *schema.Message, *model.AppState) (*schema.Message, error) {
	return func(ctx context.Context, out *schema.Message, state *model.AppState) (*schema.Message, error) {
		// Compute usage cost if available
		if model.CostEnabled() && out != nil && out.ResponseMeta != nil && out.ResponseMeta.Usage != nil {
			pricing := model.ResolvePricing(modelName)
			inC, outC, totalC := model.ComputeCost(out.ResponseMeta.Usage, pricing)
			if out.Extra == nil {
				out.Extra = map[string]any{}
			}
			out.Extra["usage_cost"] = map[string]any{
				"currency":          "USD",
				"model":             modelName,
				"prompt_tokens":     out.ResponseMeta.Usage.PromptTokens,
				"completion_tokens": out.ResponseMeta.Usage.CompletionTokens,
				"total_tokens":      out.ResponseMeta.Usage.TotalTokens,
				"input_cost":        inC,
				"output_cost":       outC,
				"total_cost":        totalC,
			}
			logx.Debug().
				Str("conversation_id", state.ConversationID).
				Str("node", NodeResponseChatModel).
				Str("model", modelName).
				Int("prompt_tokens", out.ResponseMeta.Usage.PromptTokens).
				Int("completion_tokens", out.ResponseMeta.Usage.CompletionTokens).
				Int("total_tokens", out.ResponseMeta.Usage.TotalTokens).
				Float64("input_cost_usd", inC).
				Float64("output_cost_usd", outC).
				Float64("total_cost_usd", totalC).
				Msg("LLM usage")

			// Accumulate only total cost into state
			state.TotalCostUSD += totalC
			// Also expose running total in the message Extra for visibility
			out.Extra["usage_cost_total_usd"] = state.TotalCostUSD
		}

		// Normalize tool calls: some providers (Gemini OpenAI-compat) may omit tool_call IDs.
		if out != nil && len(out.ToolCalls) > 0 {
			for i := range out.ToolCalls {
				if strings.TrimSpace(out.ToolCalls[i].ID) == "" {
					state.ToolCallIDSeq++
					out.ToolCalls[i].ID = fmt.Sprintf("call_%d", state.ToolCallIDSeq)
				}
			}
		}

		state.History = append(state.History, out)

		// Clean logging for tool calls and responses
		if len(out.ToolCalls) > 0 {
			logx.Debug().Int("tool_count", len(out.ToolCalls)).Msg("Calling tools")
		} else {
			logx.Debug().Msg("AI response ready")
		}

		// Filter and save AssistantMessage to Redis
		// Save only when it's a final assistant message (no further tool calls),
		// or when we've reached the tool-call limit but still have a content response.
		if out.Role == schema.Assistant && (len(out.ToolCalls) == 0 || state.ToolCallLimitReached) && strings.TrimSpace(out.Content) != "" {
			if err := mm.SaveResponse(ctx, state.ConversationID, out.Content); err != nil {
				logx.Error().
					Str("conversation_id", state.ConversationID).
					Err(err).
					Msg("Error saving assistant response in postHandlerResponse")
			} else {
				logx.Debug().
					Str("conversation_id", state.ConversationID).
					Msg("Successfully saved assistant response to Redis")
			}
		}

		return out, nil
	}
}

// NewToolExecutorCondition creates the condition function for tool execution routing
func NewToolExecutorCondition() func(context.Context, *schema.Message) (string, error) {
	return func(ctx context.Context, input *schema.Message) (string, error) {
		// Check if tool limit was reached
		var limitReached bool
		compose.ProcessState(ctx, func(_ context.Context, state *model.AppState) error {
			limitReached = state.ToolCallLimitReached
			return nil
		})

		if limitReached {
			logx.Debug().Msg("Tool limit reached previously - routing to end")
			return compose.END, nil
		}

		if len(input.ToolCalls) > 0 {
			logx.Debug().Int("tool_count", len(input.ToolCalls)).Msg("Routing to ToolExecutor")
			return NodeToolExecutor, nil
		}

		logx.Debug().Msg("No tool calls - continuing to end")
		return compose.END, nil
	}
}

// NewToolExecutorPreHandler creates the pre-handler for ToolExecutor node
func NewToolExecutorPreHandler(maxToolCalls int) func(context.Context, *schema.Message, *model.AppState) (*schema.Message, error) {
	return func(ctx context.Context, in *schema.Message, state *model.AppState) (*schema.Message, error) {
		// TODO: Production-grade resource management (ordered by priority)
		//
		// CRITICAL (Security & Availability):
		// 1. [HIGH] Implement per-conversation rate limiting to prevent abuse
		// 2. [HIGH] Add tool input validation and sanitization for security
		// 3. [HIGH] Implement circuit breaker pattern for external API failures
		// 4. [MEDIUM] Add tool authentication and permission validation
		//
		// PERFORMANCE (Scalability):
		// 5. [HIGH] Add exponential backoff between rapid tool calls
		// 6. [MEDIUM] Track tool execution time with configurable timeouts
		// 7. [MEDIUM] Monitor memory usage for large tool responses
		// 8. [LOW] Implement response caching for frequently used tools
		//
		// OBSERVABILITY (Monitoring):
		// 9. [HIGH] Add structured logging with correlation IDs
		// 10. [MEDIUM] Implement metrics collection (execution time, success rate)
		// 11. [MEDIUM] Add distributed tracing for tool call chains
		// 12. [LOW] Create alerting for tool failure patterns
		//
		// USER EXPERIENCE (Graceful Degradation):
		// [DONE] Basic tool call limit with graceful fallback message
		// 13. [MEDIUM] Implement partial success handling (some tools fail, others succeed)
		// 14. [LOW] Add retry mechanism with intelligent backoff for transient failures

		// Increment tool call counter
		exceeded := incrementToolCallAndCheck(state, maxToolCalls)

		logx.Debug().
			Int("tool_call_count", state.ToolCallCount).
			Str("conversation_id", state.ConversationID).
			Msg("Tool execution attempt")

		if exceeded {
			maxToolCalls = normalizeMaxToolCalls(maxToolCalls)
			logx.Warn().
				Int("tool_call_count", state.ToolCallCount).
				Int("max_tool_calls", maxToolCalls).
				Str("conversation_id", state.ConversationID).
				Msg("Tool call limit exceeded - flagging and continuing")
			return in, nil
		}

		return in, nil
	}
}

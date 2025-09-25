package model

import (
	"github.com/cloudwego/eino/schema"
)

// AppState stores per-invocation state for the Eino Graph.
// Concurrency model:
//   - This struct is registered as Graph Local State via compose.WithGenLocalState.
//   - All reads/writes happen only inside Eino state handlers:
//     WithStatePreHandler, WithStatePostHandler, or compose.ProcessState.
//   - Eino serializes access to state within these handlers, so no additional
//     mutex/atomic is required as long as you never touch it outside handlers.
//   - Do not access AppState directly from outside handlers. For persistence,
//     use repositories/services (e.g., MessagesManager).
type AppState struct {
    ConversationID       string
    History              []*schema.Message // mutated only inside Eino state handlers
    NLUAnalysis          *NLUResponse      // set by parser post-handler, read by assembler
    ToolCallCount        int               // maintained in handlers (reset/increment)
    ToolCallLimitReached bool              // set when tool call limit is exceeded
    ToolCallIDSeq        int               // local sequence to synthesize tool_call_id when provider omits

    // Accumulated total LLM cost (USD) across model invocations for this query
    TotalCostUSD float64
}

// QueryInput represents the input for processing user queries.
type QueryInput struct {
	ConversationID string `json:"conversation_id"`
	Query          string `json:"query"`
}

// ResponseData holds the data for the response.
type ResponseData struct {
	Analysis       NLUResponse // NLU analysis result
	ConversationID string      // Conversation identifier from state
}

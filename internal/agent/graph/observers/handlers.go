package observers

import (
	einocb "github.com/cloudwego/eino/callbacks"
	callbackHelper "github.com/cloudwego/eino/utils/callbacks"
)

// NewAllCallbacks aggregates all observer handlers (prompt, tool, etc.) into one callbacks.Handler.
func NewAllCallbacks() einocb.Handler {
	// Rebuild the typed handlers so we can attach them in a single helper
	toolHandler := newToolHandler()
	promptHandler := newPromptHandler()
	modelHandler := newModelHandler()

	return callbackHelper.NewHandlerHelper().
		Tool(toolHandler).
		ChatModel(modelHandler).
		Prompt(promptHandler).
		Handler()
}

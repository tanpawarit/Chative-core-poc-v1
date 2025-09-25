package observers

import (
	"context"
	"fmt"

	einocb "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/prompt"
	callbackHelper "github.com/cloudwego/eino/utils/callbacks"
)

// newPromptHandler builds a typed PromptCallbackHandler (not yet wrapped).
func newPromptHandler() *callbackHelper.PromptCallbackHandler {
	return &callbackHelper.PromptCallbackHandler{
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *prompt.CallbackInput) context.Context {
			// Keep logging generic to avoid coupling to struct fields
			fmt.Printf("[Prompt|%s|%s] start\n", info.Type, info.Name)
			if input != nil {
				fmt.Printf("input: %+v\n", input)
			}
			fmt.Println("=================================================")
			return ctx
		},
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *prompt.CallbackOutput) context.Context {
			fmt.Printf("[Prompt|%s|%s] end\n", info.Type, info.Name)
			if output != nil && len(output.Result) > 0 && output.Result[0] != nil {
				content := output.Result[0].Content
				fmt.Printf("rendered: %s\n", content)
			}
			fmt.Println("=================================================")
			return ctx
		},
		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			fmt.Printf("[Prompt|%s|%s] error: %v\n", info.Type, info.Name, err)
			fmt.Println("=================================================")
			return ctx
		},
	}
}

// NewPromptCallbacks constructs a callbacks.Handler for prompt lifecycle events.
// It logs prompt inputs and rendered outputs for observability.
func NewPromptCallbacks() einocb.Handler {
	return callbackHelper.NewHandlerHelper().
		Prompt(newPromptHandler()).
		Handler()
}

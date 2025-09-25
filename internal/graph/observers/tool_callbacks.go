package observers

import (
	"context"
	"errors"
	"fmt"
	"io"

	einocb "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	callbackHelper "github.com/cloudwego/eino/utils/callbacks"
)

// newToolHandler builds a typed ToolCallbackHandler (not yet wrapped).
func newToolHandler() *callbackHelper.ToolCallbackHandler {
	return &callbackHelper.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *tool.CallbackInput) context.Context {
			// Basic visibility for tool starts; keep stdout for now
			fmt.Printf("[TOOL START] %s input=%+v", info.Name, input.ArgumentsInJSON)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *tool.CallbackOutput) context.Context {
			fmt.Printf("[TOOL END] %s output=%+v", info.Name, output.Response)
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, info *einocb.RunInfo, output *schema.StreamReader[*tool.CallbackOutput]) context.Context {
			fmt.Println("Tool started streaming output")
			go func() {
				defer output.Close()
				for {
					chunk, err := output.Recv()
					if errors.Is(err, io.EOF) {
						return
					}
					if err != nil {
						return
					}
					fmt.Printf("Received streaming output: %s\n", chunk.Response)
				}
			}()
			return ctx
		},
		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			fmt.Printf("Tool execution failed with error: %v\n", err)
			return ctx
		},
	}
}

// NewToolCallbacks constructs a callbacks.Handler that logs tool lifecycle events.
// Attach it via compose.WithCallbacks(...) when invoking or compiling the graph.
func NewToolCallbacks() einocb.Handler {
	return callbackHelper.NewHandlerHelper().
		Tool(newToolHandler()).
		Handler()
}

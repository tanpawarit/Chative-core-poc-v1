package observers

import (
	"context"
	"fmt"
	"strings"

	einocb "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	callbackHelper "github.com/cloudwego/eino/utils/callbacks"
)

// newModelHandler builds a typed ModelCallbackHandler to log user/assistant messages around model calls.
func newModelHandler() *callbackHelper.ModelCallbackHandler {
	return &callbackHelper.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *model.CallbackInput) context.Context {
			fmt.Printf("[Model|%s|%s] start\n", info.Type, info.Name)
			// Best-effort extract the latest user message content
			if input != nil && len(input.Messages) > 0 {
				if um := lastUserContent(input.Messages); um != "" {
					fmt.Printf("user: %s\n", um)
				}
				// Log full message context (system + history)
				fmt.Println("================ context (system + history): ================")
				for i, m := range input.Messages {
					if m == nil {
						continue
					}
					role := string(m.Role)
					content := strings.TrimSpace(m.Content)
					if content == "" {
						continue
					}
					fmt.Printf("%02d %-9s: %s\n", i, role, content)
				}
			}
			fmt.Println("=================================================")
			return ctx
		},
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *model.CallbackOutput) context.Context {
			fmt.Printf("[Model|%s|%s] end\n", info.Type, info.Name)
			if output != nil && output.Message != nil {
				content := strings.TrimSpace(output.Message.Content)
				if content != "" {
					fmt.Printf("assistant: %s\n", content)
				}
			}
			fmt.Println("=================================================")
			return ctx
		},
		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			fmt.Printf("[Model|%s|%s] error: %v\n", info.Type, info.Name, err)
			fmt.Println("=================================================")
			return ctx
		},
	}
}

func lastUserContent(msgs []*schema.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m == nil {
			continue
		}
		if m.Role == schema.User {
			return strings.TrimSpace(m.Content)
		}
	}
	return ""
}

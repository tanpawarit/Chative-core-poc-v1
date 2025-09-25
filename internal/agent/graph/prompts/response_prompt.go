package prompts

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"

	"github.com/Chative-core-poc-v1/server/internal/agent/graph/tools"
	"github.com/Chative-core-poc-v1/server/internal/agent/model"
)

//go:embed template/response_prompt.txt
var coreSystemPrompt string

// RenderResponseSystem renders the dynamic Response system prompt and triggers prompt callbacks.
func RenderResponseSystem(ctx context.Context, config model.ResponsePromptConfig, nlu model.NLUResponse) (string, error) {
	// derive and normalize primary language for the template
	pl := strings.ToLower(strings.TrimSpace(nlu.PrimaryLanguage))
	if pl == "" {
		pl = "eng"
	}
	switch pl {
	case "th":
		pl = "tha"
	case "en":
		pl = "eng"
	}

	// Render via Eino prompt component (Go template) to both format and emit callbacks
	tpl := prompt.FromMessages(
		schema.GoTemplate,
		schema.SystemMessage(coreSystemPrompt),
	)
	vars := map[string]any{
		"BusinessType":    config.BusinessType,
		"BusinessName":    config.BusinessName,
		"PrimaryLanguage": pl,
		"SearchTool":      tools.ToolSearchProduct,
		"DetailsTool":     tools.ToolGetProductDetails,
	}
	msgs, err := tpl.Format(ctx, vars)
	if err != nil {
		return "", fmt.Errorf("response prompt render: %w", err)
	}
	if len(msgs) == 0 || msgs[0] == nil {
		return "", fmt.Errorf("response prompt render: empty result")
	}
	return msgs[0].Content, nil
}

package prompts

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"
)

//go:embed template/nlu_prompt.txt
var nluSystemPrompt string

// RenderNLUSystem renders the NLU system prompt via Eino prompt component.
// This triggers Prompt callbacks and returns the final system prompt string.
func RenderNLUSystem(ctx context.Context, nluConfig *model.NLUModelConfig) (string, error) {
	if nluConfig == nil {
		return "", fmt.Errorf("nlu config is nil")
	}

	// Safely render known tokens only to avoid interfering with JSON braces in template
	content := strings.NewReplacer(
		"{TD}", "<||>",
		"{RD}", "##",
		"{CD}", "<|COMPLETE|>",
		"{default_intent}", nluConfig.DefaultIntent,
		"{additional_intent}", nluConfig.AdditionalIntent,
		"{default_entity}", nluConfig.DefaultEntity,
		"{additional_entity}", nluConfig.AdditionalEntity,
	).Replace(nluSystemPrompt)

	// Wrap via Eino prompt component using a messages placeholder to emit callbacks
	tpl := prompt.FromMessages(
		schema.FString,
		schema.MessagesPlaceholder("system_messages", false),
	)
	msgs, err := tpl.Format(ctx, map[string]any{
		"system_messages": []*schema.Message{schema.SystemMessage(content)},
	})
	if err != nil {
		return "", fmt.Errorf("nlu prompt callbacks: %w", err)
	}
	if len(msgs) == 0 || msgs[0] == nil {
		return "", fmt.Errorf("nlu prompt callbacks: empty result")
	}
	return msgs[0].Content, nil
}

package conversations

import (
	"context"
	"strings"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"

	"github.com/cloudwego/eino/schema"
)

type MessagesManager struct {
    conversationRepo model.ConversationRepository
    nluMaxTurns      int
}

func NewMessagesManager(conversationRepo model.ConversationRepository, config model.ConversationConfig) *MessagesManager {
    return &MessagesManager{
        conversationRepo: conversationRepo,
        nluMaxTurns:      config.NLU.MaxTurns,
    }
}

// =========== Function for NLU ===========
func (cm *MessagesManager) ProcessNLUMessage(ctx context.Context, conversationID string, query string) (string, error) {
	// TODO: Add input validation for conversationID and query parameters
	// - Validate conversationID is not empty and follows expected format
	// - Validate query length (max 10000 chars) and sanitize input
	// - Add rate limiting per customer to prevent abuse

	// Save user message
	userMsg := schema.UserMessage(query)
	if err := cm.conversationRepo.AddMessage(ctx, conversationID, userMsg); err != nil {
		return "", err
	}

	// Load history and build context
	history, err := cm.conversationRepo.LoadHistory(ctx, conversationID)
	if err != nil {
		return "", err
	}

	conversationContext := cm.buildNLUContext(history.Messages)

	// Build complete context with current message
	var fullContext strings.Builder
	fullContext.WriteString(conversationContext)
	fullContext.WriteString("\n<current_message_to_analyze>\n")
	fullContext.WriteString("UserMessage(" + query + ")\n")
	fullContext.WriteString("</current_message_to_analyze>")

	return fullContext.String(), nil
}

func (cm *MessagesManager) buildNLUContext(messages []*schema.Message) string {
	// TODO: Add context length validation and truncation
	// - Implement max context token limit to prevent LLM overflow
	// - Add intelligent truncation that preserves important messages
	// - Consider message importance scoring for context selection

	recentMessages := trimTail(messages, cm.nluMaxTurns)

	var contextBuilder strings.Builder
	contextBuilder.WriteString("<conversation_context>\n")

	for _, msg := range recentMessages {
		if msg == nil || msg.Content == "" { // Add nil check
			continue
		}
		switch msg.Role {
		case schema.User:
			contextBuilder.WriteString("UserMessage(" + msg.Content + ")\n")
		case schema.Assistant:
			contextBuilder.WriteString("AssistantMessage(" + msg.Content + ")\n")
		}
	}

	contextBuilder.WriteString("</conversation_context>")
	return contextBuilder.String()
}

func (cm *MessagesManager) BuildResponseContext(ctx context.Context, conversationID string, systemPrompt string) ([]*schema.Message, error) {
	history, err := cm.conversationRepo.LoadHistory(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}

	messages = append(messages, history.Messages...)

	return messages, nil
}

func (cm *MessagesManager) SaveResponse(ctx context.Context, conversationID string, content string) error {
	assistantMsg := schema.AssistantMessage(content, nil)
	return cm.conversationRepo.AddMessage(ctx, conversationID, assistantMsg)
}

// ====================== Helper function ======================
func trimTail(messages []*schema.Message, maxTurns int) []*schema.Message {
	// TODO: Optimize memory allocation and improve performance
	// - Return slice reference instead of copying when possible
	// - Add bounds checking to prevent panic
	// - Consider using sync.Pool for repeated slice allocations
	// - Add metrics for memory usage tracking

	if len(messages) <= maxTurns {
		result := make([]*schema.Message, len(messages))
		copy(result, messages)
		return result
	}
	source := messages[len(messages)-maxTurns:]
	result := make([]*schema.Message, len(source))
	copy(result, source)
	return result
}

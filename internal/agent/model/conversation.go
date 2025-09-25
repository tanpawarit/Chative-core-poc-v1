package model

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type ConversationRepository interface {
	// AddMessage adds a message to the conversation history for the given conversation
	AddMessage(ctx context.Context, conversationID string, message *schema.Message) error

	// LoadHistory retrieves the conversation history for a conversation
	LoadHistory(ctx context.Context, conversationID string) (*ConversationHistory, error)

	// ClearHistory removes all conversation history for a conversation
	ClearHistory(ctx context.Context, conversationID string) error

	// GetMessageCount returns the number of messages in the conversation
	GetMessageCount(ctx context.Context, conversationID string) (int, error)
}

// ConversationHistory represents loaded conversation data with metadata.
type ConversationHistory struct {
	ConversationID string
	Messages       []*schema.Message
}

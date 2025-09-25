package nodes

import (
	"context"
	"fmt"

	logx "github.com/Chative-core-poc-v1/server/pkg/logger"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"
)

// ChatModelConfig holds the configuration for chat model creation
type ChatModelConfig struct {
	APIKey     string
	BaseURL    string
	NLUConfig  *model.NLUModelConfig
	RespConfig *model.ResponseModelConfig
}

// ChatModels holds both NLU and Response chat models
type ChatModels struct {
	NLU               *gemini.ChatModel
	Response          *gemini.ChatModel
	NLUModelName      string
	ResponseModelName string
}

// NewChatModels creates both NLU and Response chat models with the given configuration
func NewChatModels(ctx context.Context, config ChatModelConfig) (*ChatModels, error) {

	clientCfg := &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	}
	if config.BaseURL != "" {
		clientCfg.HTTPOptions.BaseURL = config.BaseURL
	}

	client, err := genai.NewClient(ctx, clientCfg)
	if err != nil {
		logx.Error().Err(err).Msg("Error creating Gemini client")
		return nil, fmt.Errorf("error creating Gemini client: %w", err)
	}

	// Create NLU Chat Model
	chatModelNLU, err := gemini.NewChatModel(ctx, &gemini.Config{
		Client:      client,
		Model:       config.NLUConfig.Model,
		Temperature: &config.NLUConfig.Temperature,
		MaxTokens:   &config.NLUConfig.MaxTokens,
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  genai.Ptr(int32(2000)),
		},
	})
	if err != nil {
		logx.Error().Err(err).Msg("Error creating NLU model")
		return nil, fmt.Errorf("error creating NLU model: %w", err)
	}

	// Create Response Chat Model
	chatModelResponse, err := gemini.NewChatModel(ctx, &gemini.Config{
		Client:      client,
		Model:       config.RespConfig.Model,
		Temperature: &config.RespConfig.Temperature,
		MaxTokens:   &config.RespConfig.MaxTokens,
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  genai.Ptr(int32(2000)),
		},
	})
	if err != nil {
		logx.Error().Err(err).Msg("Error creating Response model")
		return nil, fmt.Errorf("error creating Response model: %w", err)
	}

	return &ChatModels{
		NLU:               chatModelNLU,
		Response:          chatModelResponse,
		NLUModelName:      config.NLUConfig.Model,
		ResponseModelName: config.RespConfig.Model,
	}, nil
}

// BindToolsToResponseModel binds tools to the response chat model
func (cm *ChatModels) BindToolsToResponseModel(ctx context.Context, tools []*schema.ToolInfo) error {
	// Bind tools to model with verification
	err := cm.Response.BindTools(tools)
	if err != nil {
		logx.Error().Err(err).Msg("Failed to bind tools")
		return fmt.Errorf("failed to bind tools: %w", err)
	}

	logx.Debug().Msg("Successfully bound tools to response model")
	return nil
}

// NewNLUChatModelNode creates a wrapper for the NLU chat model to be used as a node
func NewNLUChatModelNode(chatModel *gemini.ChatModel) *gemini.ChatModel {
	return chatModel
}

// NewResponseChatModelNode creates a wrapper for the Response chat model to be used as a node
func NewResponseChatModelNode(chatModel *gemini.ChatModel) *gemini.ChatModel {
	return chatModel
}

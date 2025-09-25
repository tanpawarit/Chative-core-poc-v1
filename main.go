package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Chative-core-poc-v1/server/internal/agent/graph"
	"github.com/Chative-core-poc-v1/server/internal/agent/model"
	"github.com/Chative-core-poc-v1/server/internal/agent/repo"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
)

// AppConfig defines all configurable parameters for the agent example,
// sourced from environment variables (loaded from .env for local runs).
type AppConfig struct {
	// Infrastructure
	RedisURL string `envconfig:"REDIS_URL" required:"true"`

	// LLM provider
	APIKey  string `envconfig:"GEMINI_API_KEY" required:"true"`
	BaseURL string `envconfig:"GEMINI_BASE_URL"`

	// Agent configs
	NLU          model.NLUModelConfig
	Response     model.ResponseModelConfig
	Prompt       model.ResponsePromptConfig
	Conversation model.ConversationConfig
}

func main() {
	fmt.Println("Testing Agent Conversation Repository...")
	ctx := context.Background()
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load structured config from env
	var envCfg AppConfig
	if err := envconfig.Process("", &envCfg); err != nil {
		log.Fatalf("Failed to process environment config: %v", err)
	}

	// Parse Redis URL and create client with TLS support
	opts, err := redis.ParseURL(envCfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	rdb := redis.NewClient(opts)
	defer rdb.Close()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis successfully")

	// ====================================================
	// Build graph config entirely from env
	ttl, err := time.ParseDuration(envCfg.Conversation.TTL)
	if err != nil {
		log.Fatalf("Invalid CONVERSATION_TTL '%s': %v", envCfg.Conversation.TTL, err)
	}

	cfg := graph.Config{
		APIKey:           envCfg.APIKey,
		BaseURL:          envCfg.BaseURL,
		NLUModel:         envCfg.NLU,
		ResponseModel:    envCfg.Response,
		ResponsePrompt:   envCfg.Prompt,
		Conversation:     envCfg.Conversation,
		ConversationRepo: repo.NewRedisConversationRepository(rdb, ttl),
	}

	runner, err := graph.BuildResponseGraph(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to build graph: %v", err)
	}

	testQueries := []struct {
		description string
		query       string
	}{
		{
			description: "Initial greeting and product inquiry",
			query:       "สวัสดีครับ ผมสนใจซื้อคอมครับ",
		},
		{
			description: "Budget and feature inquiry",
			query:       "งบประมาณ 40,000 บาท ควรซื้อรุ่นไหนดีครับ ",
		},
		{
			description: "Follow-up with thanks",
			query:       "ขอบคุณครับ",
		},
		// {
		// 	description: "Purchase decision and support request",
		// 	query:       "ตกลงเอาคอม Acer แล้วกัน",
		// },
	}

	conversationID := "test-conversation-123451"

	for i, test := range testQueries {
		fmt.Printf("\n🚀 Test %d: %s\n", i+1, test.description)
		fmt.Printf("Query: \"%s\"\n", test.query)
		fmt.Println("Processing...")

		response, err := runner.Invoke(ctx, model.QueryInput{
			ConversationID: conversationID,
			Query:          test.query,
		})
		if err != nil {
			log.Fatalf("Failed to invoke graph for test %d: %v", i+1, err)
		}

		fmt.Printf("✅ Response %d: %s\n", i+1, response)
		fmt.Println("─────────────────────────────────────────────")

		// เพิ่มเวลาหยุดสั้นๆ เพื่อให้ log ชัดเจน
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("🎉 All graph tests completed successfully!")
}

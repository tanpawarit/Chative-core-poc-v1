package model

// ================ Config ================
type ConversationConfig struct {
    TTL string `envconfig:"CONVERSATION_TTL" default:"15m"`
    NLU struct {
        MaxTurns int `envconfig:"CONVERSATION_NLU_MAX_TURNS" default:"5"`
    }
    Tools struct {
        MaxCalls int `envconfig:"CONVERSATION_TOOL_MAX_CALLS" default:"10"`
    }
}

type NLUModelConfig struct {
    Model               string   `envconfig:"NLU_MODEL" default:"openai/gpt-3.5-turbo"`
    MaxTokens           int      `envconfig:"NLU_MAX_TOKENS" default:"2000"`
    Temperature         float32  `envconfig:"NLU_TEMPERATURE" default:"0.1"`
    DefaultIntent       string   `envconfig:"NLU_DEFAULT_INTENT" default:"greet:0.1, purchase_intent:0.8, inquiry_intent:0.7, support_intent:0.6, complain_intent:0.6"`
    AdditionalIntent    string   `envconfig:"NLU_ADDITIONAL_INTENT" default:"complaint:0.5, cancel_order:0.4, ask_price:0.6, compare_product:0.5, delivery_issue:0.7"`
    DefaultEntity       string   `envconfig:"NLU_DEFAULT_ENTITY" default:"product, quantity, brand, price"`
    AdditionalEntity    string   `envconfig:"NLU_ADDITIONAL_ENTITY" default:"color, model, spec, budget, warranty, delivery"`
}

type ResponseModelConfig struct {
	Model       string  `envconfig:"RESPONSE_MODEL" default:"openai/gpt-3.5-turbo"`
	MaxTokens   int     `envconfig:"RESPONSE_MAX_TOKENS" default:"2000"`
	Temperature float32 `envconfig:"RESPONSE_TEMPERATURE" default:"0.4"`
}

type ResponsePromptConfig struct {
	BusinessType string `envconfig:"PROMPT_BUSINESS_TYPE" default:"electronics store"`
	BusinessName string `envconfig:"PROMPT_BUSINESS_NAME" default:"TechHub"`
}

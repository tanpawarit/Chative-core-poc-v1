package model

import (
	"github.com/cloudwego/eino/schema"
)

// Pricing defines USD cost per 1M tokens for input/output.
type Pricing struct {
	InputPerM  float64
	OutputPerM float64
}

// defaultPricing provides hardcoded USD pricing per 1M tokens (text tokens).
var defaultPricing = map[string]Pricing{
	// Source: Gemini pricing (Standard; text). Adjust for audio/image if needed.
	"gemini-2.5-flash":      {InputPerM: 0.30, OutputPerM: 2.50},
	"gemini-2.5-flash-lite": {InputPerM: 0.10, OutputPerM: 0.40},
}

// CostEnabled returns whether to compute/log cost.
func CostEnabled() bool {
	//always enable cost computation.
	return true
}

// ResolvePricing returns hardcoded pricing for a model.
func ResolvePricing(model string) Pricing {
	var p Pricing
	var ok bool
	if p, ok = defaultPricing[model]; !ok {
		// fallback to zero pricing if unknown
		p = Pricing{}
	}
	return p
}

// ComputeCost converts token usage to USD cost using per-1M Pricing.
func ComputeCost(usage *schema.TokenUsage, p Pricing) (inputCost, outputCost, total float64) {
	if usage == nil {
		return 0, 0, 0
	}
	inputCost = p.InputPerM * float64(usage.PromptTokens) / 1_000_000.0
	outputCost = p.OutputPerM * float64(usage.CompletionTokens) / 1_000_000.0
	total = inputCost + outputCost
	return
}

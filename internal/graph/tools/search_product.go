package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// ===================================
// Search Product Tool
// ===================================

type SearchProductInput struct {
	Query      string `json:"query"`
	Category   string `json:"category,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type SearchProductOutput struct {
	Products []model.Product `json:"products"`
	Total    int             `json:"total"`
}

func createSearchProductTool() tool.BaseTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "search_product",
			Desc: "Search for products in inventory. Supports Thai/English keywords including: มือถือ, โทรศัพท์, smartphone, phone, คอมพิวเตอร์, laptop, computer, แล็ปท็อป, โน้ตบุ๊ค. Always returns structured product data with ID, name, price, and availability. Use this tool whenever customer mentions any product.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     "string",
					Desc:     "Product search keywords in Thai or English. Examples: มือถือ, smartphone, คอม, laptop, iPhone, Samsung, MacBook. Can include brand names, product types, or model numbers.",
					Required: true,
				},
				"category": {
					Type: "string",
					Desc: "Optional category filter. Available categories: smartphones, laptops, tablets, audio, wearables",
				},
				"max_results": {
					Type: "number",
					Desc: "Maximum number of products to return (default: 10, max: 20)",
				},
			}),
		},
		func(ctx context.Context, in *SearchProductInput) (*SearchProductOutput, error) {
			if in.Query == "" {
				return nil, fmt.Errorf("query is required")
			}

			if in.MaxResults == 0 {
				in.MaxResults = 10
			}

			// Search through mock products
			var matchedProducts []model.Product
			queryLower := strings.ToLower(in.Query)

			for _, product := range MockProducts {
				// Simple search matching name, category, or description
				if strings.Contains(strings.ToLower(product.Name), queryLower) ||
					strings.Contains(strings.ToLower(product.Category), queryLower) ||
					strings.Contains(strings.ToLower(product.Description), queryLower) {

					// Filter by category if specified
					if in.Category != "" && !strings.EqualFold(product.Category, in.Category) {
						continue
					}

					matchedProducts = append(matchedProducts, product)
				}
			}

			if len(matchedProducts) > in.MaxResults {
				matchedProducts = matchedProducts[:in.MaxResults]
			}

			result := &SearchProductOutput{
				Products: matchedProducts,
				Total:    len(matchedProducts),
			}

			return result, nil
		},
	)
}

var MockProducts = []model.Product{
	{
		ID:          "prod-001",
		Name:        "iPhone 15 Pro",
		Category:    "smartphones",
		Price:       39900.00,
		Description: "Latest iPhone with A17 Pro chip, titanium design, and advanced camera system โทรศัพท์ไอโฟน สมาร์ทโฟน",
		InStock:     true,
	},
	{
		ID:          "prod-002",
		Name:        "Samsung Galaxy S24 Ultra",
		Category:    "smartphones",
		Price:       42900.00,
		Description: "Premium Android phone with S Pen, 200MP camera, and AI features โทรศัพท์แอนดรอยด์ สมาร์ทโฟน",
		InStock:     true,
	},
	{
		ID:          "prod-003",
		Name:        "MacBook Air M3",
		Category:    "laptops",
		Price:       42900.00,
		Description: "Lightweight laptop with M3 chip, 13-inch Liquid Retina display โน้ตบุ๊ค แล็ปท็อป คอมพิวเตอร์พกพา งานทั่วไป",
		InStock:     false,
	},
	{
		ID:          "prod-004",
		Name:        "AirPods Pro (3rd generation)",
		Category:    "audio",
		Price:       8900.00,
		Description: "Wireless earbuds with active noise cancellation and spatial audio",
		InStock:     true,
	},
	{
		ID:          "prod-005",
		Name:        "iPad Pro 12.9-inch",
		Category:    "tablets",
		Price:       35900.00,
		Description: "Professional tablet with M2 chip and Liquid Retina XDR display",
		InStock:     true,
	},
	{
		ID:          "prod-006",
		Name:        "Sony WH-1000XM5",
		Category:    "audio",
		Price:       12900.00,
		Description: "Premium wireless headphones with industry-leading noise cancellation",
		InStock:     true,
	},
	{
		ID:          "prod-007",
		Name:        "Dell XPS 13",
		Category:    "laptops",
		Price:       35900.00,
		Description: "Premium ultrabook with Intel 13th Gen processors and InfinityEdge display โน้ตบุ๊ค แล็ปท็อป อัลตร้าบุ๊ค งานทั่วไป",
		InStock:     true,
	},
	{
		ID:          "prod-008",
		Name:        "Apple Watch Ultra 2",
		Category:    "wearables",
		Price:       29900.00,
		Description: "Rugged smartwatch for outdoor adventures with precise GPS นาฬิกาอัจฉริยะ สมาร์ทวอช",
		InStock:     false,
	},
	// เพิ่มโน้ตบุ๊คสำหรับงบประมาณ 30,000 บาท
	{
		ID:          "prod-009",
		Name:        "Acer Aspire 5 A515-58",
		Category:    "laptops",
		Price:       28900.00,
		Description: "Budget laptop Intel Core i5, 8GB RAM, 512GB SSD สำหรับงานทั่วไป โน้ตบุ๊ค แล็ปท็อป คอมพิวเตอร์พกพา ราคาประหยัด งานทั่วไป เล่นเกม",
		InStock:     true,
	},
	{
		ID:          "prod-010",
		Name:        "Lenovo IdeaPad 3 Gaming",
		Category:    "laptops",
		Price:       29500.00,
		Description: "Gaming laptop AMD Ryzen 5, 8GB RAM, GTX 1650 สำหรับเล่นเกม โน้ตบุ๊ค แล็ปท็อป เกมมิ่ง งานทั่วไป เล่นเกม",
		InStock:     true,
	},
	{
		ID:          "prod-011",
		Name:        "HP Pavilion 15-eh3000",
		Category:    "laptops",
		Price:       27900.00,
		Description: "All-purpose laptop AMD Ryzen 5, 8GB RAM, 256GB SSD สำหรับงานทั่วไปและเล่นเกมเบาๆ โน้ตบุ๊ค แล็ปท็อป คอมพิวเตอร์พกพา งานทั่วไป เล่นเกม",
		InStock:     true,
	},
	{
		ID:          "prod-012",
		Name:        "ASUS VivoBook 15 X1502ZA",
		Category:    "laptops",
		Price:       24900.00,
		Description: "Affordable laptop Intel Core i3, 8GB RAM, 512GB SSD สำหรับงานเบาๆ โน้ตบุ๊ค แล็ปท็อป ราคาประหยัด งานทั่วไป",
		InStock:     true,
	},
}

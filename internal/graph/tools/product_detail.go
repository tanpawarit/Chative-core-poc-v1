package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type GetProductDetailsInput struct {
	ProductID string `json:"product_id"`
}

type GetProductDetailsOutput struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Price          float64           `json:"price"`
	Specifications map[string]string `json:"specifications"`
	InStock        bool              `json:"in_stock"`
}

func createGetProductDetailsTool() tool.BaseTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "get_product_details",
			Desc: "Get comprehensive product specifications and details. Returns complete technical specifications, features, availability status, and descriptions. Use this tool when customer needs detailed product information or comparisons.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"product_id": {
					Type:     "string",
					Desc:     "Product ID obtained from search_product results (e.g., prod-001, prod-002). Must be exact ID from search results.",
					Required: true,
				},
			}),
		},
		func(ctx context.Context, in *GetProductDetailsInput) (*GetProductDetailsOutput, error) {
			if in.ProductID == "" {
				return nil, fmt.Errorf("product_id is required")
			}

			// Look up product details from mock data
			if details, exists := MockProductDetails[in.ProductID]; exists {
				return &details, nil
			}

			// If not found in detailed mock data, try to find in basic product data
			for _, product := range MockProducts {
				if product.ID == in.ProductID {
					result := &GetProductDetailsOutput{
						ID:          product.ID,
						Name:        product.Name,
						Description: product.Description,
						Price:       product.Price,
						Specifications: map[string]string{
							"category": product.Category,
							"in_stock": fmt.Sprintf("%v", product.InStock),
						},
						InStock: product.InStock,
					}
					return result, nil
				}
			}

			return nil, fmt.Errorf("product not found: %s", in.ProductID)
		},
	)
}

var MockProductDetails = map[string]GetProductDetailsOutput{
	"prod-001": {
		ID:          "prod-001",
		Name:        "iPhone 15 Pro",
		Description: "The iPhone 15 Pro features a titanium design, A17 Pro chip with 6-core GPU, advanced camera system with 48MP main camera, and USB-C connectivity.",
		Price:       39900.00,
		Specifications: map[string]string{
			"display":      "6.1-inch Super Retina XDR",
			"chip":         "A17 Pro",
			"storage":      "128GB, 256GB, 512GB, 1TB",
			"camera":       "48MP Main, 12MP Ultra Wide, 12MP Telephoto",
			"battery":      "Up to 23 hours video playback",
			"connectivity": "5G, WiFi 6E, Bluetooth 5.3",
			"color":        "Natural Titanium, Blue Titanium, White Titanium, Black Titanium",
		},
		InStock: true,
	},
	"prod-002": {
		ID:          "prod-002",
		Name:        "Samsung Galaxy S24 Ultra",
		Description: "Premium flagship with S Pen, 200MP camera, AI-powered features, and titanium frame for ultimate productivity and creativity.",
		Price:       42900.00,
		Specifications: map[string]string{
			"display":   "6.8-inch Dynamic AMOLED 2X",
			"processor": "Snapdragon 8 Gen 3",
			"storage":   "256GB, 512GB, 1TB",
			"camera":    "200MP Wide, 50MP Periscope Telephoto, 10MP Telephoto, 12MP Ultra Wide",
			"battery":   "5000mAh with 45W fast charging",
			"s_pen":     "Built-in S Pen with Air Actions",
			"color":     "Titanium Gray, Titanium Black, Titanium Violet, Titanium Yellow",
		},
		InStock: true,
	},
	"prod-003": {
		ID:          "prod-003",
		Name:        "MacBook Air M3",
		Description: "The new MacBook Air with M3 chip delivers exceptional performance and battery life in an incredibly thin and light design.",
		Price:       42900.00,
		Specifications: map[string]string{
			"display": "13.6-inch Liquid Retina",
			"chip":    "Apple M3 with 8-core CPU and 10-core GPU",
			"memory":  "8GB, 16GB, 24GB unified memory",
			"storage": "256GB, 512GB, 1TB, 2TB SSD",
			"battery": "Up to 18 hours",
			"ports":   "2x Thunderbolt / USB 4, 3.5mm headphone jack, MagSafe 3",
			"color":   "Space Gray, Silver, Starlight, Midnight",
		},
		InStock: false,
	},
	"prod-009": {
		ID:          "prod-009",
		Name:        "Acer Aspire 5 A515-58",
		Description: "Budget-friendly laptop perfect for everyday tasks and light gaming. Features Intel Core i5 processor, 8GB RAM, and 512GB SSD storage.",
		Price:       28900.00,
		Specifications: map[string]string{
			"display":   "15.6-inch Full HD IPS",
			"processor": "Intel Core i5-1235U",
			"memory":    "8GB DDR4 RAM",
			"storage":   "512GB NVMe SSD",
			"graphics":  "Intel Iris Xe Graphics",
			"battery":   "Up to 8 hours",
			"color":     "Silver, Black",
		},
		InStock: true,
	},
	"prod-010": {
		ID:          "prod-010",
		Name:        "Lenovo IdeaPad 3 Gaming",
		Description: "Gaming laptop with AMD Ryzen 5 processor and dedicated graphics card. Perfect for gaming and multimedia tasks.",
		Price:       29500.00,
		Specifications: map[string]string{
			"display":   "15.6-inch Full HD 120Hz",
			"processor": "AMD Ryzen 5 5500H",
			"memory":    "8GB DDR4 RAM",
			"storage":   "512GB NVMe SSD",
			"graphics":  "NVIDIA GTX 1650 4GB",
			"battery":   "Up to 6 hours",
			"color":     "Shadow Black",
		},
		InStock: true,
	},
	"prod-011": {
		ID:          "prod-011",
		Name:        "HP Pavilion 15-eh3000",
		Description: "Versatile laptop for work and light entertainment. AMD Ryzen 5 processor with good performance and battery life.",
		Price:       27900.00,
		Specifications: map[string]string{
			"display":   "15.6-inch Full HD IPS",
			"processor": "AMD Ryzen 5 5625U",
			"memory":    "8GB DDR4 RAM",
			"storage":   "256GB NVMe SSD",
			"graphics":  "AMD Radeon Graphics",
			"battery":   "Up to 9 hours",
			"color":     "Natural Silver, Warm Gold",
		},
		InStock: true,
	},
}

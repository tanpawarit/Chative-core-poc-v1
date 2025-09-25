package model

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
}

type ProductPrice struct {
	ProductID     string  `json:"product_id"`
	CurrentPrice  float64 `json:"current_price"`
	OriginalPrice float64 `json:"original_price"`
	Discount      float64 `json:"discount"`
}

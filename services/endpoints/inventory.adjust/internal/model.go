package internal

type AdjustRequest struct {
	SKU   string `json:"sku"`
	Delta int    `json:"delta"`
}

type AdjustResponse struct {
	SKU      string `json:"sku"`
	NewCount int    `json:"new_count"`
}

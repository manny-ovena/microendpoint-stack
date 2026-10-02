package internal

import "errors"

func ValidateAdjustRequest(req AdjustRequest) error {
	if req.SKU == "" {
		return errors.New("sku is required")
	}
	if req.Delta == 0 {
		return errors.New("delta must be non-zero")
	}
	return nil
}

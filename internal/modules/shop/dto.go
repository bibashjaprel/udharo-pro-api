package shop

type CurrentShopResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Phone        *string `json:"phone"`
	Address      *string `json:"address"`
	BusinessType *string `json:"business_type"`
	LogoURL      *string `json:"logo_url"`
	Status       string  `json:"status"`
}

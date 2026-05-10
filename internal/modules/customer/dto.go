package customer

import "time"

type CreateCustomerRequest struct {
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Address *string `json:"address"`
	Notes   *string `json:"notes"`
}

type CustomerResponse struct {
	ID        int64     `json:"id"`
	ShopID    int64     `json:"shop_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Address   *string   `json:"address"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

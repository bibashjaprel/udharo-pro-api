package customer

import "time"

type CreateCustomerRequest struct {
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Address *string `json:"address"`
	Notes   *string `json:"notes"`
}

type ListCustomersRequest struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

type ListCustomersResponse struct {
	Customers []CustomerResponse `json:"customers"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
	Total     int64              `json:"total"`
}

type CustomerDetailsResponse struct {
	CustomerResponse
	CurrentBalance float64 `json:"current_balance"`
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

package ledger

import "time"

type CreateCreditEntryRequest struct {
	Amount          float64 `json:"amount"`
	Note            *string `json:"note"`
	TransactionDate string  `json:"transaction_date"`
}

type ListLedgerEntriesRequest struct {
	Page             int
	Limit            int
	IncludeCancelled bool
}

type CustomerLedgerStatementResponse struct {
	CustomerID     int64                 `json:"customer_id"`
	ShopID         int64                 `json:"shop_id"`
	Entries        []LedgerEntryResponse `json:"entries"`
	Page           int                   `json:"page"`
	Limit          int                   `json:"limit"`
	Total          int64                 `json:"total"`
	CurrentBalance float64               `json:"current_balance"`
}

type LedgerEntryResponse struct {
	ID              int64     `json:"id"`
	ShopID          int64     `json:"shop_id"`
	CustomerID      int64     `json:"customer_id"`
	EntryType       string    `json:"entry_type"`
	Amount          float64   `json:"amount"`
	Note            *string   `json:"note"`
	TransactionDate time.Time `json:"transaction_date"`
	Status          string    `json:"status"`
	CreatedBy       int64     `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

package ledger

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EntryTypeCredit     = "credit"
	EntryTypePayment    = "payment"
	EntryTypeAdjustment = "adjustment"
)

var (
	ErrInvalidCreditEntry = errors.New("invalid credit entry")
	ErrInvalidPagination  = errors.New("invalid pagination")
	ErrCustomerNotFound   = errors.New("customer not found")
)

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repository: NewRepository(db)}
}

func (s *Service) CreateCreditEntry(ctx context.Context, userID int64, shopID int64, customerID int64, req CreateCreditEntryRequest) (LedgerEntryResponse, error) {
	if customerID < 1 {
		return LedgerEntryResponse{}, ErrCustomerNotFound
	}

	entry, err := normalizeCreateCreditEntryRequest(req)
	if err != nil {
		return LedgerEntryResponse{}, err
	}

	return s.repository.CreateCreditEntry(ctx, userID, shopID, customerID, entry)
}

func (s *Service) ListCustomerLedger(ctx context.Context, shopID int64, customerID int64, req ListLedgerEntriesRequest) (CustomerLedgerStatementResponse, error) {
	if customerID < 1 {
		return CustomerLedgerStatementResponse{}, ErrCustomerNotFound
	}

	req = normalizeListLedgerEntriesRequest(req)
	if err := validateListLedgerEntriesRequest(req); err != nil {
		return CustomerLedgerStatementResponse{}, err
	}

	return s.repository.ListCustomerLedger(ctx, shopID, customerID, req)
}

type createCreditEntryFields struct {
	EntryType       string
	Amount          float64
	Note            *string
	TransactionDate time.Time
}

func normalizeCreateCreditEntryRequest(req CreateCreditEntryRequest) (createCreditEntryFields, error) {
	transactionDate, err := parseRequiredTransactionDate(req.TransactionDate)
	if err != nil {
		return createCreditEntryFields{}, err
	}

	fields := createCreditEntryFields{
		EntryType:       EntryTypeCredit,
		Amount:          req.Amount,
		Note:            optionalString(req.Note),
		TransactionDate: transactionDate,
	}
	if err := validateLedgerEntryFields(fields); err != nil {
		return createCreditEntryFields{}, err
	}

	return fields, nil
}

func normalizeListLedgerEntriesRequest(req ListLedgerEntriesRequest) ListLedgerEntriesRequest {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	return req
}

func validateListLedgerEntriesRequest(req ListLedgerEntriesRequest) error {
	if req.Page < 1 || req.Limit < 1 || req.Limit > 100 {
		return ErrInvalidPagination
	}

	return nil
}

func parseRequiredTransactionDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, ErrInvalidCreditEntry
	}

	transactionDate, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, ErrInvalidCreditEntry
	}

	return transactionDate, nil
}

func validateLedgerEntryFields(fields createCreditEntryFields) error {
	if !validEntryType(fields.EntryType) {
		return ErrInvalidCreditEntry
	}
	if fields.Amount <= 0 || math.IsNaN(fields.Amount) || math.IsInf(fields.Amount, 0) {
		return ErrInvalidCreditEntry
	}
	if fields.TransactionDate.IsZero() {
		return ErrInvalidCreditEntry
	}

	return nil
}

func validEntryType(entryType string) bool {
	switch entryType {
	case EntryTypeCredit, EntryTypePayment, EntryTypeAdjustment:
		return true
	default:
		return false
	}
}

func optionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

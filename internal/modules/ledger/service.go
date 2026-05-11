package ledger

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCreditEntry = errors.New("invalid credit entry")
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

type createCreditEntryFields struct {
	Amount          float64
	Note            *string
	TransactionDate time.Time
}

func normalizeCreateCreditEntryRequest(req CreateCreditEntryRequest) (createCreditEntryFields, error) {
	if req.Amount <= 0 {
		return createCreditEntryFields{}, ErrInvalidCreditEntry
	}

	transactionDate := time.Now().UTC()
	if strings.TrimSpace(req.TransactionDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.TransactionDate))
		if err != nil {
			return createCreditEntryFields{}, ErrInvalidCreditEntry
		}
		transactionDate = parsed
	}

	return createCreditEntryFields{
		Amount:          req.Amount,
		Note:            optionalString(req.Note),
		TransactionDate: transactionDate,
	}, nil
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

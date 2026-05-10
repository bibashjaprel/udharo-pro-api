package customer

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCustomer        = errors.New("invalid customer")
	ErrDuplicateCustomerPhone = errors.New("customer phone already exists")
)

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repository: NewRepository(db)}
}

func (s *Service) CreateCustomer(ctx context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error) {
	req = normalizeCreateCustomerRequest(req)
	if err := validateCreateCustomerRequest(req); err != nil {
		return CustomerResponse{}, err
	}

	return s.repository.CreateCustomer(ctx, userID, shopID, req)
}

func normalizeCreateCustomerRequest(req CreateCustomerRequest) CreateCustomerRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Address = optionalString(req.Address)
	req.Notes = optionalString(req.Notes)
	return req
}

func validateCreateCustomerRequest(req CreateCustomerRequest) error {
	if req.Name == "" || req.Phone == "" {
		return ErrInvalidCustomer
	}

	return nil
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

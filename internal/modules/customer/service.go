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
	ErrInvalidPagination      = errors.New("invalid pagination")
	ErrCustomerNotFound       = errors.New("customer not found")
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

func (s *Service) UpdateCustomer(ctx context.Context, userID int64, shopID int64, customerID int64, req UpdateCustomerRequest) (CustomerResponse, error) {
	if customerID < 1 {
		return CustomerResponse{}, ErrCustomerNotFound
	}

	fields, err := normalizeUpdateCustomerRequest(req)
	if err != nil {
		return CustomerResponse{}, err
	}

	return s.repository.UpdateCustomer(ctx, userID, shopID, customerID, fields)
}

func (s *Service) DeleteCustomer(ctx context.Context, userID int64, shopID int64, customerID int64) error {
	if customerID < 1 {
		return ErrCustomerNotFound
	}

	return s.repository.DeleteCustomer(ctx, userID, shopID, customerID)
}

func (s *Service) ListCustomers(ctx context.Context, shopID int64, req ListCustomersRequest) (ListCustomersResponse, error) {
	req = normalizeListCustomersRequest(req)
	if err := validateListCustomersRequest(req); err != nil {
		return ListCustomersResponse{}, err
	}

	return s.repository.ListCustomers(ctx, shopID, req)
}

func (s *Service) GetCustomer(ctx context.Context, shopID int64, customerID int64) (CustomerDetailsResponse, error) {
	if customerID < 1 {
		return CustomerDetailsResponse{}, ErrCustomerNotFound
	}

	return s.repository.GetCustomer(ctx, shopID, customerID)
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

type updateCustomerFields struct {
	NameSet    bool
	Name       string
	PhoneSet   bool
	Phone      string
	AddressSet bool
	Address    *string
	NotesSet   bool
	Notes      *string
}

func normalizeUpdateCustomerRequest(req UpdateCustomerRequest) (updateCustomerFields, error) {
	fields := updateCustomerFields{}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return updateCustomerFields{}, ErrInvalidCustomer
		}
		fields.NameSet = true
		fields.Name = name
	}

	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		if phone == "" {
			return updateCustomerFields{}, ErrInvalidCustomer
		}
		fields.PhoneSet = true
		fields.Phone = phone
	}

	if req.Address != nil {
		fields.AddressSet = true
		fields.Address = optionalString(req.Address)
	}

	if req.Notes != nil {
		fields.NotesSet = true
		fields.Notes = optionalString(req.Notes)
	}

	if !fields.NameSet && !fields.PhoneSet && !fields.AddressSet && !fields.NotesSet {
		return updateCustomerFields{}, ErrInvalidCustomer
	}

	return fields, nil
}

func normalizeListCustomersRequest(req ListCustomersRequest) ListCustomersRequest {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	req.Search = strings.TrimSpace(req.Search)
	return req
}

func validateListCustomersRequest(req ListCustomersRequest) error {
	if req.Page < 1 || req.Limit < 1 || req.Limit > 100 {
		return ErrInvalidPagination
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

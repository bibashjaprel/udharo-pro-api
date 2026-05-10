package shop

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrShopNotFound        = errors.New("shop not found")
	ErrInvalidShopProfile  = errors.New("invalid shop profile")
	ErrShopUpdateForbidden = errors.New("shop update forbidden")
)

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repository: NewRepository(db)}
}

func (s *Service) CurrentShop(ctx context.Context, shopID int64) (CurrentShopResponse, error) {
	return s.repository.FindCurrentShop(ctx, shopID)
}

func (s *Service) UpdateShop(ctx context.Context, userID int64, shopID int64, role string, req UpdateShopRequest) (CurrentShopResponse, error) {
	if role != "owner" {
		return CurrentShopResponse{}, ErrShopUpdateForbidden
	}

	fields, err := normalizeUpdateShopRequest(req)
	if err != nil {
		return CurrentShopResponse{}, err
	}

	return s.repository.UpdateCurrentShop(ctx, userID, shopID, fields)
}

type updateShopFields struct {
	NameSet         bool
	Name            string
	PhoneSet        bool
	Phone           *string
	AddressSet      bool
	Address         *string
	BusinessTypeSet bool
	BusinessType    *string
	LogoURLSet      bool
	LogoURL         *string
}

func normalizeUpdateShopRequest(req UpdateShopRequest) (updateShopFields, error) {
	fields := updateShopFields{}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return updateShopFields{}, ErrInvalidShopProfile
		}
		fields.NameSet = true
		fields.Name = name
	}

	if req.Phone != nil {
		fields.PhoneSet = true
		fields.Phone = optionalString(*req.Phone)
	}
	if req.Address != nil {
		fields.AddressSet = true
		fields.Address = optionalString(*req.Address)
	}
	if req.BusinessType != nil {
		fields.BusinessTypeSet = true
		fields.BusinessType = optionalString(*req.BusinessType)
	}
	if req.LogoURL != nil {
		fields.LogoURLSet = true
		fields.LogoURL = optionalString(*req.LogoURL)
	}

	if !fields.NameSet && !fields.PhoneSet && !fields.AddressSet && !fields.BusinessTypeSet && !fields.LogoURLSet {
		return updateShopFields{}, ErrInvalidShopProfile
	}

	return fields, nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

package shop

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrShopNotFound = errors.New("shop not found")

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repository: NewRepository(db)}
}

func (s *Service) CurrentShop(ctx context.Context, shopID int64) (CurrentShopResponse, error) {
	return s.repository.FindCurrentShop(ctx, shopID)
}

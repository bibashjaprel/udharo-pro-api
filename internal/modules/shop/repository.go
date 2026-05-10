package shop

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindCurrentShop(ctx context.Context, shopID int64) (CurrentShopResponse, error) {
	var res CurrentShopResponse
	var phone sql.NullString
	var address sql.NullString
	var businessType sql.NullString
	var logoURL sql.NullString

	err := r.db.QueryRow(ctx, `
		SELECT id, name, phone, address, business_type, logo_url, status
		FROM shops
		WHERE id = $1
			AND deleted_at IS NULL
		LIMIT 1
	`, shopID).Scan(
		&res.ID,
		&res.Name,
		&phone,
		&address,
		&businessType,
		&logoURL,
		&res.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CurrentShopResponse{}, ErrShopNotFound
	}
	if err != nil {
		return CurrentShopResponse{}, fmt.Errorf("find current shop: %w", err)
	}

	res.Phone = nullableString(phone)
	res.Address = nullableString(address)
	res.BusinessType = nullableString(businessType)
	res.LogoURL = nullableString(logoURL)

	return res, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

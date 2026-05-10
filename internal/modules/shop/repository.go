package shop

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (r *Repository) UpdateCurrentShop(ctx context.Context, userID int64, shopID int64, fields updateShopFields) (CurrentShopResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CurrentShopResponse{}, fmt.Errorf("begin shop update transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var res CurrentShopResponse
	var phone sql.NullString
	var address sql.NullString
	var businessType sql.NullString
	var logoURL sql.NullString

	err = tx.QueryRow(ctx, `
		UPDATE shops
		SET
			name = CASE WHEN $2 THEN $3 ELSE name END,
			phone = CASE WHEN $4 THEN $5 ELSE phone END,
			address = CASE WHEN $6 THEN $7 ELSE address END,
			business_type = CASE WHEN $8 THEN $9 ELSE business_type END,
			logo_url = CASE WHEN $10 THEN $11 ELSE logo_url END,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING id, name, phone, address, business_type, logo_url, status
	`,
		shopID,
		fields.NameSet,
		fields.Name,
		fields.PhoneSet,
		fields.Phone,
		fields.AddressSet,
		fields.Address,
		fields.BusinessTypeSet,
		fields.BusinessType,
		fields.LogoURLSet,
		fields.LogoURL,
	).Scan(
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
		return CurrentShopResponse{}, fmt.Errorf("update current shop: %w", err)
	}

	res.Phone = nullableString(phone)
	res.Address = nullableString(address)
	res.BusinessType = nullableString(businessType)
	res.LogoURL = nullableString(logoURL)

	metadata, err := json.Marshal(map[string]any{
		"updated_fields": updatedShopFieldNames(fields),
	})
	if err != nil {
		return CurrentShopResponse{}, fmt.Errorf("encode shop update audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (shop_id, user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, shopID, userID, "shop.updated", "shop", shopID, metadata)
	if err != nil {
		return CurrentShopResponse{}, fmt.Errorf("create shop update audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CurrentShopResponse{}, fmt.Errorf("commit shop update transaction: %w", err)
	}

	return res, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func updatedShopFieldNames(fields updateShopFields) []string {
	names := make([]string, 0, 5)
	if fields.NameSet {
		names = append(names, "name")
	}
	if fields.PhoneSet {
		names = append(names, "phone")
	}
	if fields.AddressSet {
		names = append(names, "address")
	}
	if fields.BusinessTypeSet {
		names = append(names, "business_type")
	}
	if fields.LogoURLSet {
		names = append(names, "logo_url")
	}

	return names
}

package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCustomer(ctx context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("begin customer create transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var res CustomerResponse
	var address sql.NullString
	var notes sql.NullString

	err = tx.QueryRow(ctx, `
		INSERT INTO customers (shop_id, name, phone, address, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, shop_id, name, phone, address, notes, created_at, updated_at
	`, shopID, req.Name, req.Phone, req.Address, req.Notes).Scan(
		&res.ID,
		&res.ShopID,
		&res.Name,
		&res.Phone,
		&address,
		&notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		return CustomerResponse{}, mapCreateCustomerDBError(err)
	}

	res.Address = nullableString(address)
	res.Notes = nullableString(notes)

	metadata, err := json.Marshal(map[string]any{
		"customer_name":  res.Name,
		"customer_phone": res.Phone,
	})
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("encode customer create audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (shop_id, user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, shopID, userID, "customer.created", "customer", res.ID, metadata)
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("create customer audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CustomerResponse{}, fmt.Errorf("commit customer create transaction: %w", err)
	}

	return res, nil
}

func mapCreateCustomerDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_customers_shop_id_phone_unique" {
		return ErrDuplicateCustomerPhone
	}

	return fmt.Errorf("create customer: %w", err)
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

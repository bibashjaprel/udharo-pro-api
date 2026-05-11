package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) ListCustomers(ctx context.Context, shopID int64, req ListCustomersRequest) (ListCustomersResponse, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM customers
		WHERE shop_id = $1
			AND deleted_at IS NULL
			AND (
				$2 = ''
				OR name ILIKE '%' || $2 || '%'
				OR phone ILIKE '%' || $2 || '%'
				OR address ILIKE '%' || $2 || '%'
			)
	`, shopID, req.Search).Scan(&total); err != nil {
		return ListCustomersResponse{}, fmt.Errorf("count customers: %w", err)
	}

	offset := (req.Page - 1) * req.Limit
	rows, err := r.db.Query(ctx, `
		SELECT id, shop_id, name, phone, address, notes, created_at, updated_at
		FROM customers
		WHERE shop_id = $1
			AND deleted_at IS NULL
			AND (
				$2 = ''
				OR name ILIKE '%' || $2 || '%'
				OR phone ILIKE '%' || $2 || '%'
				OR address ILIKE '%' || $2 || '%'
			)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, shopID, req.Search, req.Limit, offset)
	if err != nil {
		return ListCustomersResponse{}, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	customers := make([]CustomerResponse, 0)
	for rows.Next() {
		var customer CustomerResponse
		var address sql.NullString
		var notes sql.NullString

		if err := rows.Scan(
			&customer.ID,
			&customer.ShopID,
			&customer.Name,
			&customer.Phone,
			&address,
			&notes,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		); err != nil {
			return ListCustomersResponse{}, fmt.Errorf("scan customer: %w", err)
		}

		customer.Address = nullableString(address)
		customer.Notes = nullableString(notes)
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return ListCustomersResponse{}, fmt.Errorf("iterate customers: %w", err)
	}

	return ListCustomersResponse{
		Customers: customers,
		Page:      req.Page,
		Limit:     req.Limit,
		Total:     total,
	}, nil
}

func (r *Repository) UpdateCustomer(ctx context.Context, userID int64, shopID int64, customerID int64, fields updateCustomerFields) (CustomerResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("begin customer update transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	oldCustomer, err := r.getCustomerForUpdate(ctx, tx, shopID, customerID)
	if err != nil {
		return CustomerResponse{}, err
	}

	var res CustomerResponse
	var address sql.NullString
	var notes sql.NullString

	err = tx.QueryRow(ctx, `
		UPDATE customers
		SET
			name = CASE WHEN $3 THEN $4 ELSE name END,
			phone = CASE WHEN $5 THEN $6 ELSE phone END,
			address = CASE WHEN $7 THEN $8 ELSE address END,
			notes = CASE WHEN $9 THEN $10 ELSE notes END,
			updated_at = NOW()
		WHERE id = $1
			AND shop_id = $2
			AND deleted_at IS NULL
		RETURNING id, shop_id, name, phone, address, notes, created_at, updated_at
	`,
		customerID,
		shopID,
		fields.NameSet,
		fields.Name,
		fields.PhoneSet,
		fields.Phone,
		fields.AddressSet,
		fields.Address,
		fields.NotesSet,
		fields.Notes,
	).Scan(
		&res.ID,
		&res.ShopID,
		&res.Name,
		&res.Phone,
		&address,
		&notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerResponse{}, ErrCustomerNotFound
	}
	if err != nil {
		return CustomerResponse{}, mapCustomerPhoneDBError(err, "update customer")
	}

	res.Address = nullableString(address)
	res.Notes = nullableString(notes)

	metadata, err := json.Marshal(map[string]any{
		"old_values": customerAuditValues(oldCustomer),
		"new_values": customerAuditValues(res),
	})
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("encode customer update audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (shop_id, user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, shopID, userID, "customer.updated", "customer", res.ID, metadata)
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("create customer update audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CustomerResponse{}, fmt.Errorf("commit customer update transaction: %w", err)
	}

	return res, nil
}

func (r *Repository) GetCustomer(ctx context.Context, shopID int64, customerID int64) (CustomerDetailsResponse, error) {
	var res CustomerDetailsResponse
	var address sql.NullString
	var notes sql.NullString

	err := r.db.QueryRow(ctx, `
		SELECT id, shop_id, name, phone, address, notes, created_at, updated_at
		FROM customers
		WHERE id = $1
			AND shop_id = $2
			AND deleted_at IS NULL
	`, customerID, shopID).Scan(
		&res.ID,
		&res.ShopID,
		&res.Name,
		&res.Phone,
		&address,
		&notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerDetailsResponse{}, ErrCustomerNotFound
	}
	if err != nil {
		return CustomerDetailsResponse{}, fmt.Errorf("get customer: %w", err)
	}

	res.Address = nullableString(address)
	res.Notes = nullableString(notes)
	res.CurrentBalance = 0

	return res, nil
}

type customerQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) getCustomerForUpdate(ctx context.Context, q customerQuerier, shopID int64, customerID int64) (CustomerResponse, error) {
	var res CustomerResponse
	var address sql.NullString
	var notes sql.NullString

	err := q.QueryRow(ctx, `
		SELECT id, shop_id, name, phone, address, notes, created_at, updated_at
		FROM customers
		WHERE id = $1
			AND shop_id = $2
			AND deleted_at IS NULL
		FOR UPDATE
	`, customerID, shopID).Scan(
		&res.ID,
		&res.ShopID,
		&res.Name,
		&res.Phone,
		&address,
		&notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerResponse{}, ErrCustomerNotFound
	}
	if err != nil {
		return CustomerResponse{}, fmt.Errorf("get customer for update: %w", err)
	}

	res.Address = nullableString(address)
	res.Notes = nullableString(notes)

	return res, nil
}

func mapCreateCustomerDBError(err error) error {
	return mapCustomerPhoneDBError(err, "create customer")
}

func mapCustomerPhoneDBError(err error, operation string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_customers_shop_id_phone_unique" {
		return ErrDuplicateCustomerPhone
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func customerAuditValues(customer CustomerResponse) map[string]any {
	return map[string]any{
		"name":    customer.Name,
		"phone":   customer.Phone,
		"address": customer.Address,
		"notes":   customer.Notes,
	}
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

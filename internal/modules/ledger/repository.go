package ledger

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

func (r *Repository) CreateCreditEntry(ctx context.Context, userID int64, shopID int64, customerID int64, fields createCreditEntryFields) (LedgerEntryResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("begin credit entry transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := ensureCustomerBelongsToShop(ctx, tx, shopID, customerID); err != nil {
		return LedgerEntryResponse{}, err
	}

	var res LedgerEntryResponse
	var note sql.NullString

	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (shop_id, customer_id, entry_type, amount, note, transaction_date, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, shop_id, customer_id, entry_type, amount, note, transaction_date, status, created_by, created_at, updated_at
	`, shopID, customerID, fields.EntryType, fields.Amount, fields.Note, fields.TransactionDate, "active", userID).Scan(
		&res.ID,
		&res.ShopID,
		&res.CustomerID,
		&res.EntryType,
		&res.Amount,
		&note,
		&res.TransactionDate,
		&res.Status,
		&res.CreatedBy,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("create credit entry: %w", err)
	}

	res.Note = nullableString(note)

	metadata, err := json.Marshal(map[string]any{
		"customer_id":      customerID,
		"entry_type":       res.EntryType,
		"amount":           res.Amount,
		"transaction_date": res.TransactionDate.Format("2006-01-02"),
	})
	if err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("encode credit entry audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (shop_id, user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, shopID, userID, "ledger.credit.created", "ledger_entry", res.ID, metadata)
	if err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("create credit entry audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("commit credit entry transaction: %w", err)
	}

	return res, nil
}

func (r *Repository) ListCustomerLedger(ctx context.Context, shopID int64, customerID int64, req ListLedgerEntriesRequest) (CustomerLedgerStatementResponse, error) {
	if err := ensureCustomerBelongsToShop(ctx, r.db, shopID, customerID); err != nil {
		return CustomerLedgerStatementResponse{}, err
	}

	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ledger_entries
		WHERE shop_id = $1
			AND customer_id = $2
			AND deleted_at IS NULL
			AND ($3 OR status <> 'cancelled')
	`, shopID, customerID, req.IncludeCancelled).Scan(&total); err != nil {
		return CustomerLedgerStatementResponse{}, fmt.Errorf("count customer ledger entries: %w", err)
	}

	offset := (req.Page - 1) * req.Limit
	rows, err := r.db.Query(ctx, `
		SELECT id, shop_id, customer_id, entry_type, amount, note, transaction_date, status, created_by, created_at, updated_at
		FROM ledger_entries
		WHERE shop_id = $1
			AND customer_id = $2
			AND deleted_at IS NULL
			AND ($3 OR status <> 'cancelled')
		ORDER BY transaction_date DESC, id DESC
		LIMIT $4 OFFSET $5
	`, shopID, customerID, req.IncludeCancelled, req.Limit, offset)
	if err != nil {
		return CustomerLedgerStatementResponse{}, fmt.Errorf("list customer ledger entries: %w", err)
	}
	defer rows.Close()

	entries := make([]LedgerEntryResponse, 0)
	for rows.Next() {
		entry, err := scanLedgerEntry(rows)
		if err != nil {
			return CustomerLedgerStatementResponse{}, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return CustomerLedgerStatementResponse{}, fmt.Errorf("iterate customer ledger entries: %w", err)
	}

	balance, err := calculateCustomerBalance(ctx, r.db, shopID, customerID)
	if err != nil {
		return CustomerLedgerStatementResponse{}, err
	}

	return CustomerLedgerStatementResponse{
		CustomerID:     customerID,
		ShopID:         shopID,
		Entries:        entries,
		Page:           req.Page,
		Limit:          req.Limit,
		Total:          total,
		CurrentBalance: balance,
	}, nil
}

type customerChecker interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ledgerEntryScanner interface {
	Scan(dest ...any) error
}

func ensureCustomerBelongsToShop(ctx context.Context, q customerChecker, shopID int64, customerID int64) error {
	var id int64
	err := q.QueryRow(ctx, `
		SELECT id
		FROM customers
		WHERE id = $1
			AND shop_id = $2
			AND deleted_at IS NULL
	`, customerID, shopID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCustomerNotFound
	}
	if err != nil {
		return fmt.Errorf("ensure customer belongs to shop: %w", err)
	}

	return nil
}

func calculateCustomerBalance(ctx context.Context, q customerChecker, shopID int64, customerID int64) (float64, error) {
	var balance float64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			CASE entry_type
				WHEN 'credit' THEN amount
				WHEN 'payment' THEN -amount
				WHEN 'adjustment' THEN amount
				ELSE 0
			END
		), 0)
		FROM ledger_entries
		WHERE shop_id = $1
			AND customer_id = $2
			AND status = 'active'
			AND deleted_at IS NULL
	`, shopID, customerID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("calculate customer balance: %w", err)
	}

	return balance, nil
}

func scanLedgerEntry(row ledgerEntryScanner) (LedgerEntryResponse, error) {
	var entry LedgerEntryResponse
	var note sql.NullString

	if err := row.Scan(
		&entry.ID,
		&entry.ShopID,
		&entry.CustomerID,
		&entry.EntryType,
		&entry.Amount,
		&note,
		&entry.TransactionDate,
		&entry.Status,
		&entry.CreatedBy,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return LedgerEntryResponse{}, fmt.Errorf("scan ledger entry: %w", err)
	}

	entry.Note = nullableString(note)
	return entry, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

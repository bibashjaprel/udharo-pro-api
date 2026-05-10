package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) CreateSignup(ctx context.Context, req SignupRequest, passwordHash string, trialDuration time.Duration) (SignupResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("begin signup transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, phone, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.Name, req.Email, req.Phone, passwordHash).Scan(&userID)
	if err != nil {
		return SignupResponse{}, mapSignupDBError(err)
	}

	var shopID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO shops (name, owner_id)
		VALUES ($1, $2)
		RETURNING id
	`, req.ShopName, userID).Scan(&shopID)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("create shop: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO shop_users (shop_id, user_id, role)
		VALUES ($1, $2, $3)
	`, shopID, userID, "owner")
	if err != nil {
		return SignupResponse{}, fmt.Errorf("create shop user: %w", err)
	}

	now := time.Now().UTC()
	trialEndsAt := now.Add(trialDuration)

	var subscriptionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO subscriptions (shop_id, plan_name, status, trial_starts_at, trial_ends_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, shopID, "trial", "trial", now, trialEndsAt).Scan(&subscriptionID)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("create trial subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return SignupResponse{}, fmt.Errorf("commit signup transaction: %w", err)
	}

	return SignupResponse{
		UserID:         userID,
		ShopID:         shopID,
		SubscriptionID: subscriptionID,
		Role:           "owner",
	}, nil
}

func (r *Repository) FindLoginSession(ctx context.Context, identifier string) (loginSession, string, error) {
	var session loginSession
	var passwordHash string

	err := r.db.QueryRow(ctx, `
		SELECT
			u.id,
			u.name,
			u.email,
			u.phone,
			u.status,
			u.password_hash,
			sh.id,
			sh.name,
			sh.status,
			su.role
		FROM users u
		JOIN shop_users su ON su.user_id = u.id
		JOIN shops sh ON sh.id = su.shop_id
		WHERE (u.email = $1 OR u.phone = $1)
			AND u.deleted_at IS NULL
			AND sh.deleted_at IS NULL
		ORDER BY sh.id ASC
		LIMIT 1
	`, identifier).Scan(
		&session.User.ID,
		&session.User.Name,
		&session.User.Email,
		&session.User.Phone,
		&session.User.Status,
		&passwordHash,
		&session.Shop.ID,
		&session.Shop.Name,
		&session.Shop.Status,
		&session.Shop.Role,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return loginSession{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return loginSession{}, "", fmt.Errorf("find login user: %w", err)
	}

	return session, passwordHash, nil
}

func (r *Repository) CreateUserSession(ctx context.Context, userID int64, shopID int64, tokenID string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_sessions (user_id, shop_id, token_id, expires_at)
		VALUES ($1, $2, $3, $4)
	`, userID, shopID, tokenID, expiresAt)
	if err != nil {
		return fmt.Errorf("create user session: %w", err)
	}

	return nil
}

func (r *Repository) RevokeSession(ctx context.Context, tokenID string, userID int64, shopID int64) error {
	commandTag, err := r.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = $1
		WHERE token_id = $2
			AND user_id = $3
			AND shop_id = $4
			AND revoked_at IS NULL
	`, time.Now().UTC(), tokenID, userID, shopID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *Repository) IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error) {
	var active bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_sessions
			WHERE token_id = $1
				AND user_id = $2
				AND shop_id = $3
				AND revoked_at IS NULL
				AND expires_at > $4
		)
	`, tokenID, userID, shopID, time.Now().UTC()).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("check active session: %w", err)
	}

	return active, nil
}

func mapSignupDBError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return fmt.Errorf("create user: %w", err)
	}

	switch pgErr.ConstraintName {
	case "users_email_key":
		return ErrDuplicateEmail
	case "users_phone_key":
		return ErrDuplicatePhone
	default:
		return ErrDuplicateSignup
	}
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const trialDuration = 14 * 24 * time.Hour

var (
	ErrInvalidInput    = errors.New("invalid signup input")
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrDuplicatePhone  = errors.New("phone already exists")
	ErrDuplicateSignup = errors.New("signup already exists")
)

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	ShopName string `json:"shop_name"`
}

type SignupResponse struct {
	UserID         int64  `json:"user_id"`
	ShopID         int64  `json:"shop_id"`
	SubscriptionID int64  `json:"subscription_id"`
	Role           string `json:"role"`
}

type Service struct {
	db        *pgxpool.Pool
	jwtSecret string
}

func NewService(db *pgxpool.Pool, jwtSecret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (SignupResponse, error) {
	req = normalizeSignupRequest(req)
	if err := validateSignupRequest(req); err != nil {
		return SignupResponse{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
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
	`, req.Name, req.Email, req.Phone, string(passwordHash)).Scan(&userID)
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

func normalizeSignupRequest(req SignupRequest) SignupRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	req.ShopName = strings.TrimSpace(req.ShopName)
	return req
}

func validateSignupRequest(req SignupRequest) error {
	if req.Name == "" || req.Email == "" || req.Phone == "" || req.Password == "" || req.ShopName == "" {
		return ErrInvalidInput
	}

	return nil
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

type SignupService interface {
	Signup(ctx context.Context, req SignupRequest) (SignupResponse, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
}

type Handler struct {
	service SignupService
}

func NewHandler(service SignupService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req SignupRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	res, err := h.service.Signup(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, email, phone, password, and shop_name are required"})
		case errors.Is(err, ErrDuplicateEmail):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already exists"})
		case errors.Is(err, ErrDuplicatePhone):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "phone already exists"})
		case errors.Is(err, ErrDuplicateSignup):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "signup already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "signup failed"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

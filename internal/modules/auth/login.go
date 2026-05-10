package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const accessTokenDuration = 24 * time.Hour

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginRequest struct {
	Identifier   string `json:"identifier"`
	EmailOrPhone string `json:"email_or_phone"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
}

type LoginResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
	User        LoginUserInfo `json:"user"`
	Shop        LoginShopInfo `json:"shop"`
}

type LoginUserInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

type LoginShopInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Role   string `json:"role"`
}

type accessTokenClaims struct {
	UserID int64  `json:"user_id"`
	ShopID int64  `json:"shop_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	identifier := normalizeLoginIdentifier(req)
	if identifier == "" || strings.TrimSpace(req.Password) == "" {
		return LoginResponse{}, ErrInvalidCredentials
	}

	session, passwordHash, err := s.findLoginSession(ctx, identifier)
	if err != nil {
		return LoginResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	tokenID, err := newTokenID()
	if err != nil {
		return LoginResponse{}, fmt.Errorf("create token id: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(accessTokenDuration)
	accessToken, err := s.signAccessToken(session, tokenID, now, expiresAt)
	if err != nil {
		return LoginResponse{}, err
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO user_sessions (user_id, shop_id, token_id, expires_at)
		VALUES ($1, $2, $3, $4)
	`, session.User.ID, session.Shop.ID, tokenID, expiresAt)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("create user session: %w", err)
	}

	return LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        session.User,
		Shop:        session.Shop,
	}, nil
}

type loginSession struct {
	User LoginUserInfo
	Shop LoginShopInfo
}

func (s *Service) findLoginSession(ctx context.Context, identifier string) (loginSession, string, error) {
	var session loginSession
	var passwordHash string

	err := s.db.QueryRow(ctx, `
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

func (s *Service) signAccessToken(session loginSession, tokenID string, issuedAt time.Time, expiresAt time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims{
		UserID: session.User.ID,
		ShopID: session.Shop.ID,
		Role:   session.Shop.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", session.User.ID),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	accessToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return accessToken, nil
}

func normalizeLoginIdentifier(req LoginRequest) string {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.EmailOrPhone)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Phone)
	}
	if strings.Contains(identifier, "@") {
		identifier = strings.ToLower(identifier)
	}

	return identifier
}

func newTokenID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	res, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login failed"})
		}
		return
	}

	writeJSON(w, http.StatusOK, res)
}

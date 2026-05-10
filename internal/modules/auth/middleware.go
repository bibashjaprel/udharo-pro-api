package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const (
	userIDContextKey  contextKey = "user_id"
	shopIDContextKey  contextKey = "shop_id"
	roleContextKey    contextKey = "role"
	tokenIDContextKey contextKey = "token_id"
)

type SessionValidator interface {
	IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error)
}

func AuthMiddleware(jwtSecret string, sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			claims, err := parseAccessToken(tokenString, jwtSecret)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			active, err := sessionValidator.IsSessionActive(r.Context(), claims.ID, claims.UserID, claims.ShopID)
			if err != nil || !active {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, shopIDContextKey, claims.ShopID)
			ctx = context.WithValue(ctx, roleContextKey, claims.Role)
			ctx = context.WithValue(ctx, tokenIDContextKey, claims.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok
}

func ShopIDFromContext(ctx context.Context) (int64, bool) {
	shopID, ok := ctx.Value(shopIDContextKey).(int64)
	return shopID, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleContextKey).(string)
	return role, ok
}

func TokenIDFromContext(ctx context.Context) (string, bool) {
	tokenID, ok := ctx.Value(tokenIDContextKey).(string)
	return tokenID, ok
}

type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error) {
	var active bool
	err := s.db.QueryRow(ctx, `
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

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("missing bearer token")
	}

	return parts[1], nil
}

func parseAccessToken(tokenString string, jwtSecret string) (*accessTokenClaims, error) {
	claims := &accessTokenClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("invalid signing method")
			}

			return []byte(jwtSecret), nil
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.ID == "" || claims.UserID == 0 || claims.ShopID == 0 {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

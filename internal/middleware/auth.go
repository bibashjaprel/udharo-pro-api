package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bibashjaprel/udharo-pro-api/internal/modules/auth"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type Authenticator func(http.Handler) http.Handler

type SessionValidator interface {
	IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error)
}

func Auth(jwtSecret string, sessionValidator SessionValidator) Authenticator {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
				return
			}

			claims, err := auth.ParseAccessToken(tokenString, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
				return
			}

			active, err := sessionValidator.IsSessionActive(r.Context(), claims.ID, claims.UserID, claims.ShopID)
			if err != nil || !active {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
				return
			}

			ctx := contextx.WithUserID(r.Context(), strconv.FormatInt(claims.UserID, 10))
			ctx = contextx.WithShopID(ctx, strconv.FormatInt(claims.ShopID, 10))
			ctx = contextx.WithRole(ctx, claims.Role)
			ctx = contextx.WithTokenID(ctx, claims.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("missing bearer token")
	}

	return parts[1], nil
}

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type LogoutResponse struct {
	Message string `json:"message"`
}

func (s *Service) Logout(ctx context.Context, tokenID string, userID int64, shopID int64) error {
	commandTag, err := s.db.Exec(ctx, `
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

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tokenID, ok := TokenIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	shopID, ok := ShopIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.Logout(r.Context(), tokenID, userID, shopID); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "logout failed"})
		}
		return
	}

	writeJSON(w, http.StatusOK, LogoutResponse{Message: "logged out successfully"})
}

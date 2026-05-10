package shop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CurrentShopService interface {
	CurrentShop(ctx context.Context, shopID int64) (CurrentShopResponse, error)
	UpdateShop(ctx context.Context, userID int64, shopID int64, role string, req UpdateShopRequest) (CurrentShopResponse, error)
}

type Handler struct {
	service CurrentShopService
}

func NewHandler(service CurrentShopService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CurrentShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	shopID, ok := contextx.GetShopIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	res, err := h.service.CurrentShop(r.Context(), shopID)
	if err != nil {
		switch {
		case errors.Is(err, ErrShopNotFound):
			response.Error(w, http.StatusNotFound, "shop not found", "shop not found")
		default:
			response.Error(w, http.StatusInternalServerError, "shop fetch failed", "shop fetch failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "current shop fetched successfully", res)
}

func (h *Handler) UpdateShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	userID, ok := contextx.GetUserIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	shopID, ok := contextx.GetShopIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	role, ok := contextx.GetRole(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req UpdateShopRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.UpdateShop(r.Context(), userID, shopID, role, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShopProfile):
			response.Error(w, http.StatusBadRequest, "invalid request", "invalid shop profile")
		case errors.Is(err, ErrShopUpdateForbidden):
			response.Error(w, http.StatusForbidden, "forbidden", "owner role is required")
		case errors.Is(err, ErrShopNotFound):
			response.Error(w, http.StatusNotFound, "shop not found", "shop not found")
		default:
			response.Error(w, http.StatusInternalServerError, "shop update failed", "shop update failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "shop updated successfully", res)
}

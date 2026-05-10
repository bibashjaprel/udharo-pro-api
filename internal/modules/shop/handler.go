package shop

import (
	"context"
	"errors"
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CurrentShopService interface {
	CurrentShop(ctx context.Context, shopID int64) (CurrentShopResponse, error)
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

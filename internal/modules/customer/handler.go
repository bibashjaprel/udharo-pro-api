package customer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error)
}

type Handler struct {
	service CustomerService
}

func NewHandler(service CustomerService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
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

	var req CreateCustomerRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.CreateCustomer(r.Context(), userID, shopID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCustomer):
			response.Error(w, http.StatusBadRequest, "invalid request", "name and phone are required")
		case errors.Is(err, ErrDuplicateCustomerPhone):
			response.Error(w, http.StatusConflict, "customer already exists", "customer phone already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "customer create failed", "customer create failed")
		}
		return
	}

	response.Success(w, http.StatusCreated, "customer created successfully", res)
}

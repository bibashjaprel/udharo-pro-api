package customer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error)
	ListCustomers(ctx context.Context, shopID int64, req ListCustomersRequest) (ListCustomersResponse, error)
}

type Handler struct {
	service CustomerService
}

func NewHandler(service CustomerService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Customers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListCustomers(w, r)
	case http.MethodPost:
		h.CreateCustomer(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
	}
}

func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
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

	req, err := listCustomersRequestFromQuery(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid pagination")
		return
	}

	res, err := h.service.ListCustomers(r.Context(), shopID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPagination):
			response.Error(w, http.StatusBadRequest, "invalid request", "invalid pagination")
		default:
			response.Error(w, http.StatusInternalServerError, "customers fetch failed", "customers fetch failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "customers fetched successfully", res)
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

func listCustomersRequestFromQuery(r *http.Request) (ListCustomersRequest, error) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		return ListCustomersRequest{}, err
	}

	limit, err := queryInt(r, "limit", 20)
	if err != nil {
		return ListCustomersRequest{}, err
	}

	return ListCustomersRequest{Page: page, Limit: limit}, nil
}

func queryInt(r *http.Request, key string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

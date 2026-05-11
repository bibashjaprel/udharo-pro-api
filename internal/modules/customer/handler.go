package customer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error)
	UpdateCustomer(ctx context.Context, userID int64, shopID int64, customerID int64, req UpdateCustomerRequest) (CustomerResponse, error)
	ListCustomers(ctx context.Context, shopID int64, req ListCustomersRequest) (ListCustomersResponse, error)
	GetCustomer(ctx context.Context, shopID int64, customerID int64) (CustomerDetailsResponse, error)
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

func (h *Handler) CustomerDetails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetCustomer(w, r)
	case http.MethodPatch:
		h.UpdateCustomer(w, r)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
	}
}

func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	shopID, ok := contextx.GetShopIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	customerID, err := customerIDFromPath(r.URL.Path)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid customer id")
		return
	}

	res, err := h.service.GetCustomer(r.Context(), shopID, customerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCustomerNotFound):
			response.Error(w, http.StatusNotFound, "customer not found", "customer not found")
		default:
			response.Error(w, http.StatusInternalServerError, "customer fetch failed", "customer fetch failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "customer fetched successfully", res)
}

func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
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

	customerID, err := customerIDFromPath(r.URL.Path)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid customer id")
		return
	}

	var req UpdateCustomerRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.UpdateCustomer(r.Context(), userID, shopID, customerID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCustomer):
			response.Error(w, http.StatusBadRequest, "invalid request", "invalid customer")
		case errors.Is(err, ErrCustomerNotFound):
			response.Error(w, http.StatusNotFound, "customer not found", "customer not found")
		case errors.Is(err, ErrDuplicateCustomerPhone):
			response.Error(w, http.StatusConflict, "customer already exists", "customer phone already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "customer update failed", "customer update failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "customer updated successfully", res)
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

	return ListCustomersRequest{
		Page:   page,
		Limit:  limit,
		Search: r.URL.Query().Get("search"),
	}, nil
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

func customerIDFromPath(path string) (int64, error) {
	id := strings.TrimPrefix(path, CustomersPath+"/")
	if id == "" || strings.Contains(id, "/") {
		return 0, strconv.ErrSyntax
	}

	customerID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || customerID < 1 {
		return 0, strconv.ErrSyntax
	}

	return customerID, nil
}

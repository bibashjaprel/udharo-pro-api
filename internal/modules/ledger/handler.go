package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bibashjaprel/udharo-pro-api/internal/modules/customer"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type CreditEntryService interface {
	CreateCreditEntry(ctx context.Context, userID int64, shopID int64, customerID int64, req CreateCreditEntryRequest) (LedgerEntryResponse, error)
	ListCustomerLedger(ctx context.Context, shopID int64, customerID int64, req ListLedgerEntriesRequest) (CustomerLedgerStatementResponse, error)
}

type Handler struct {
	service CreditEntryService
}

func NewHandler(service CreditEntryService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCreditEntry(w http.ResponseWriter, r *http.Request) {
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

	customerID, err := customerIDFromCreditPath(r.URL.Path)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid customer id")
		return
	}

	var req CreateCreditEntryRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.CreateCreditEntry(r.Context(), userID, shopID, customerID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCreditEntry):
			response.Error(w, http.StatusBadRequest, "invalid request", "invalid credit entry")
		case errors.Is(err, ErrCustomerNotFound):
			response.Error(w, http.StatusNotFound, "customer not found", "customer not found")
		default:
			response.Error(w, http.StatusInternalServerError, "credit entry create failed", "credit entry create failed")
		}
		return
	}

	response.Success(w, http.StatusCreated, "credit entry created successfully", res)
}

func (h *Handler) ListCustomerLedger(w http.ResponseWriter, r *http.Request) {
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

	customerID, err := customerIDFromLedgerPath(r.URL.Path)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid customer id")
		return
	}

	req, err := listLedgerEntriesRequestFromQuery(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid pagination")
		return
	}

	res, err := h.service.ListCustomerLedger(r.Context(), shopID, customerID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPagination):
			response.Error(w, http.StatusBadRequest, "invalid request", "invalid pagination")
		case errors.Is(err, ErrCustomerNotFound):
			response.Error(w, http.StatusNotFound, "customer not found", "customer not found")
		default:
			response.Error(w, http.StatusInternalServerError, "customer ledger fetch failed", "customer ledger fetch failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "customer ledger fetched successfully", res)
}

func IsCreditPath(path string) bool {
	return strings.HasPrefix(path, customer.CustomersPath+"/") && strings.HasSuffix(path, "/credit")
}

func IsLedgerPath(path string) bool {
	return strings.HasPrefix(path, customer.CustomersPath+"/") && strings.HasSuffix(path, "/ledger")
}

func customerIDFromCreditPath(path string) (int64, error) {
	id := strings.TrimPrefix(path, customer.CustomersPath+"/")
	id = strings.TrimSuffix(id, "/credit")
	return customerIDFromTrimmedPath(id)
}

func customerIDFromLedgerPath(path string) (int64, error) {
	id := strings.TrimPrefix(path, customer.CustomersPath+"/")
	id = strings.TrimSuffix(id, "/ledger")
	return customerIDFromTrimmedPath(id)
}

func customerIDFromTrimmedPath(id string) (int64, error) {
	if id == "" || strings.Contains(id, "/") {
		return 0, strconv.ErrSyntax
	}

	customerID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || customerID < 1 {
		return 0, strconv.ErrSyntax
	}

	return customerID, nil
}

func listLedgerEntriesRequestFromQuery(r *http.Request) (ListLedgerEntriesRequest, error) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		return ListLedgerEntriesRequest{}, err
	}

	limit, err := queryInt(r, "limit", 20)
	if err != nil {
		return ListLedgerEntriesRequest{}, err
	}

	includeCancelled, err := queryBool(r, "include_cancelled")
	if err != nil {
		return ListLedgerEntriesRequest{}, err
	}

	return ListLedgerEntriesRequest{
		Page:             page,
		Limit:            limit,
		IncludeCancelled: includeCancelled,
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

func queryBool(r *http.Request, key string) (bool, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return parsed, nil
}

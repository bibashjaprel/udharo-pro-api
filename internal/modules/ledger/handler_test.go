package ledger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bibashjaprel/udharo-pro-api/internal/modules/customer"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
)

type fakeCreditEntryService struct {
	response          LedgerEntryResponse
	statementResponse CustomerLedgerStatementResponse
	err               error
	statementErr      error
	userID            int64
	shopID            int64
	customerID        int64
	request           CreateCreditEntryRequest
	statementRequest  ListLedgerEntriesRequest
	called            bool
	statementCalled   bool
}

func (s *fakeCreditEntryService) CreateCreditEntry(_ context.Context, userID int64, shopID int64, customerID int64, req CreateCreditEntryRequest) (LedgerEntryResponse, error) {
	s.called = true
	s.userID = userID
	s.shopID = shopID
	s.customerID = customerID
	s.request = req
	return s.response, s.err
}

func (s *fakeCreditEntryService) ListCustomerLedger(_ context.Context, shopID int64, customerID int64, req ListLedgerEntriesRequest) (CustomerLedgerStatementResponse, error) {
	s.statementCalled = true
	s.shopID = shopID
	s.customerID = customerID
	s.statementRequest = req
	return s.statementResponse, s.statementErr
}

func TestListCustomerLedgerHandlerListsEntries(t *testing.T) {
	now := time.Now().UTC()
	service := &fakeCreditEntryService{
		statementResponse: CustomerLedgerStatementResponse{
			CustomerID:     5,
			ShopID:         2,
			Page:           2,
			Limit:          5,
			Total:          12,
			CurrentBalance: 1250,
			Entries: []LedgerEntryResponse{
				{
					ID:              7,
					ShopID:          2,
					CustomerID:      5,
					EntryType:       "credit",
					Amount:          1500,
					TransactionDate: now,
					Status:          "active",
					CreatedBy:       1,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger?page=2&limit=5&include_cancelled=true", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.statementCalled {
		t.Fatal("expected statement service to be called")
	}
	if service.shopID != 2 || service.customerID != 5 {
		t.Fatalf("unexpected args: shop=%d customer=%d", service.shopID, service.customerID)
	}
	if service.statementRequest.Page != 2 || service.statementRequest.Limit != 5 || !service.statementRequest.IncludeCancelled {
		t.Fatalf("unexpected statement request: %+v", service.statementRequest)
	}

	var response struct {
		Success bool                            `json:"success"`
		Data    CustomerLedgerStatementResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Total != 12 || response.Data.CurrentBalance != 1250 || len(response.Data.Entries) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestListCustomerLedgerHandlerDefaultsPagination(t *testing.T) {
	service := &fakeCreditEntryService{}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if service.statementRequest.Page != 1 || service.statementRequest.Limit != 20 || service.statementRequest.IncludeCancelled {
		t.Fatalf("expected default pagination, got %+v", service.statementRequest)
	}
}

func TestListCustomerLedgerHandlerRequiresTenant(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger", nil)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestListCustomerLedgerHandlerRejectsInvalidQuery(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger?page=abc", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListCustomerLedgerHandlerRejectsInvalidID(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/abc/ledger", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListCustomerLedgerHandlerHandlesInvalidPagination(t *testing.T) {
	service := &fakeCreditEntryService{statementErr: ErrInvalidPagination}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger?page=0", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListCustomerLedgerHandlerReturnsNotFound(t *testing.T) {
	service := &fakeCreditEntryService{statementErr: ErrCustomerNotFound}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/ledger", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestListCustomerLedgerHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})
	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/ledger", nil)
	rec := httptest.NewRecorder()

	handler.ListCustomerLedger(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, rec.Header().Get("Allow"))
	}
}

func TestCreateCreditEntryHandlerCreatesEntry(t *testing.T) {
	now := time.Now().UTC()
	note := "Rice, oil, sugar"
	service := &fakeCreditEntryService{
		response: LedgerEntryResponse{
			ID:              7,
			ShopID:          2,
			CustomerID:      5,
			EntryType:       "credit",
			Amount:          1500,
			Note:            &note,
			TransactionDate: now,
			Status:          "active",
			CreatedBy:       1,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	body := []byte(`{"amount":1500,"note":"Rice, oil, sugar","transaction_date":"2026-05-09"}`)
	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !service.called {
		t.Fatal("expected credit entry service to be called")
	}
	if service.userID != 1 || service.shopID != 2 || service.customerID != 5 {
		t.Fatalf("unexpected args: user=%d shop=%d customer=%d", service.userID, service.shopID, service.customerID)
	}
	if service.request.Amount != 1500 || service.request.TransactionDate != "2026-05-09" {
		t.Fatalf("unexpected request: %+v", service.request)
	}

	var response struct {
		Success bool                `json:"success"`
		Data    LedgerEntryResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.ID != 7 || response.Data.EntryType != "credit" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCreateCreditEntryHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})
	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader([]byte(`{"amount":1500}`)))
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateCreditEntryHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})
	req := httptest.NewRequest(http.MethodGet, customer.CustomersPath+"/5/credit", nil)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, rec.Header().Get("Allow"))
	}
}

func TestCreateCreditEntryHandlerRejectsInvalidID(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/abc/credit", bytes.NewReader([]byte(`{"amount":1500}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateCreditEntryHandlerRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(&fakeCreditEntryService{})

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader([]byte(`{"amount":1500,"unknown":"value"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateCreditEntryHandlerHandlesInvalidCreditEntry(t *testing.T) {
	service := &fakeCreditEntryService{err: ErrInvalidCreditEntry}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader([]byte(`{"amount":0}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateCreditEntryHandlerReturnsNotFound(t *testing.T) {
	service := &fakeCreditEntryService{err: ErrCustomerNotFound}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader([]byte(`{"amount":1500}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestCreateCreditEntryHandlerReturnsServerError(t *testing.T) {
	service := &fakeCreditEntryService{err: errors.New("database unavailable")}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, customer.CustomersPath+"/5/credit", bytes.NewReader([]byte(`{"amount":1500}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCreditEntry(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestIsCreditPath(t *testing.T) {
	if !IsCreditPath(customer.CustomersPath + "/5/credit") {
		t.Fatal("expected credit path to match")
	}
	if IsCreditPath(customer.CustomersPath + "/5") {
		t.Fatal("expected customer detail path not to match")
	}
}

func TestIsLedgerPath(t *testing.T) {
	if !IsLedgerPath(customer.CustomersPath + "/5/ledger") {
		t.Fatal("expected ledger path to match")
	}
	if IsLedgerPath(customer.CustomersPath + "/5/credit") {
		t.Fatal("expected credit path not to match")
	}
}

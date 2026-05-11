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
	response   LedgerEntryResponse
	err        error
	userID     int64
	shopID     int64
	customerID int64
	request    CreateCreditEntryRequest
	called     bool
}

func (s *fakeCreditEntryService) CreateCreditEntry(_ context.Context, userID int64, shopID int64, customerID int64, req CreateCreditEntryRequest) (LedgerEntryResponse, error) {
	s.called = true
	s.userID = userID
	s.shopID = shopID
	s.customerID = customerID
	s.request = req
	return s.response, s.err
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

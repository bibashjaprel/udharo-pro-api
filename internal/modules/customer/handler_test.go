package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
)

type fakeCustomerService struct {
	response     CustomerResponse
	listResponse ListCustomersResponse
	err          error
	listErr      error
	userID       int64
	shopID       int64
	request      CreateCustomerRequest
	listRequest  ListCustomersRequest
	called       bool
	listCalled   bool
}

func (s *fakeCustomerService) CreateCustomer(_ context.Context, userID int64, shopID int64, req CreateCustomerRequest) (CustomerResponse, error) {
	s.called = true
	s.userID = userID
	s.shopID = shopID
	s.request = req
	return s.response, s.err
}

func (s *fakeCustomerService) ListCustomers(_ context.Context, shopID int64, req ListCustomersRequest) (ListCustomersResponse, error) {
	s.listCalled = true
	s.shopID = shopID
	s.listRequest = req
	return s.listResponse, s.listErr
}

func TestListCustomersHandlerListsCustomers(t *testing.T) {
	now := time.Now().UTC()
	service := &fakeCustomerService{
		listResponse: ListCustomersResponse{
			Customers: []CustomerResponse{
				{
					ID:        1,
					ShopID:    2,
					Name:      "Ram Bahadur",
					Phone:     "9841000000",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			Page:  2,
			Limit: 5,
			Total: 12,
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CustomersPath+"?page=2&limit=5&search=ram", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Customers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.listCalled {
		t.Fatal("expected customer list service to be called")
	}
	if service.shopID != 2 {
		t.Fatalf("unexpected shop id: %d", service.shopID)
	}
	if service.listRequest.Page != 2 || service.listRequest.Limit != 5 || service.listRequest.Search != "ram" {
		t.Fatalf("unexpected list request: %+v", service.listRequest)
	}

	var response struct {
		Success bool                  `json:"success"`
		Data    ListCustomersResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Total != 12 || len(response.Data.Customers) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestListCustomersHandlerDefaultsPagination(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CustomersPath, nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if service.listRequest.Page != 1 || service.listRequest.Limit != 20 {
		t.Fatalf("expected default pagination, got %+v", service.listRequest)
	}
}

func TestListCustomersHandlerRequiresTenant(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodGet, CustomersPath, nil)
	rec := httptest.NewRecorder()

	handler.ListCustomers(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestListCustomersHandlerRejectsInvalidQuery(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CustomersPath+"?page=abc", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListCustomersHandlerRejectsInvalidPagination(t *testing.T) {
	service := &fakeCustomerService{listErr: ErrInvalidPagination}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CustomersPath+"?page=0&limit=5", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListCustomers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCustomersHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodDelete, CustomersPath, nil)
	rec := httptest.NewRecorder()

	handler.Customers(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("expected Allow header %q, got %q", "GET, POST", rec.Header().Get("Allow"))
	}
}

func TestCreateCustomerHandlerCreatesCustomer(t *testing.T) {
	address := "Kathmandu"
	notes := "Regular customer"
	now := time.Now().UTC()
	service := &fakeCustomerService{
		response: CustomerResponse{
			ID:        1,
			ShopID:    2,
			Name:      "Ram Bahadur",
			Phone:     "9841000000",
			Address:   &address,
			Notes:     &notes,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	body := []byte(`{"name":"Ram Bahadur","phone":"9841000000","address":"Kathmandu","notes":"Regular customer"}`)
	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !service.called {
		t.Fatal("expected customer service to be called")
	}
	if service.userID != 1 || service.shopID != 2 {
		t.Fatalf("unexpected create args: user=%d shop=%d", service.userID, service.shopID)
	}
	if service.request.Name != "Ram Bahadur" || service.request.Phone != "9841000000" {
		t.Fatalf("unexpected request: %+v", service.request)
	}

	var response struct {
		Success bool             `json:"success"`
		Data    CustomerResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.ID != 1 || response.Data.ShopID != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCreateCustomerHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader([]byte(`{"name":"Ram","phone":"9841000000"}`)))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateCustomerHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodGet, CustomersPath, nil)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, rec.Header().Get("Allow"))
	}
}

func TestCreateCustomerHandlerRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader([]byte(`{"name":"Ram","phone":"9841000000","unknown":"value"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateCustomerHandlerRejectsInvalidCustomer(t *testing.T) {
	service := &fakeCustomerService{err: ErrInvalidCustomer}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader([]byte(`{"name":"","phone":"9841000000"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateCustomerHandlerHandlesDuplicatePhone(t *testing.T) {
	service := &fakeCustomerService{err: ErrDuplicateCustomerPhone}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader([]byte(`{"name":"Ram","phone":"9841000000"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestCreateCustomerHandlerReturnsServerError(t *testing.T) {
	service := &fakeCustomerService{err: errors.New("database unavailable")}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodPost, CustomersPath, bytes.NewReader([]byte(`{"name":"Ram","phone":"9841000000"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

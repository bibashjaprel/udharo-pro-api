package shop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
)

type fakeCurrentShopService struct {
	response CurrentShopResponse
	err      error
	shopID   int64
	called   bool
}

func (s *fakeCurrentShopService) CurrentShop(_ context.Context, shopID int64) (CurrentShopResponse, error) {
	s.called = true
	s.shopID = shopID
	return s.response, s.err
}

func TestCurrentShopHandlerReturnsCurrentShop(t *testing.T) {
	phone := "9841000000"
	address := "Kathmandu"
	businessType := "retail"
	logoURL := "https://example.com/logo.png"
	service := &fakeCurrentShopService{
		response: CurrentShopResponse{
			ID:           2,
			Name:         "Bibas Kirana Pasal",
			Phone:        &phone,
			Address:      &address,
			BusinessType: &businessType,
			LogoURL:      &logoURL,
			Status:       "active",
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CurrentShopPath, nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CurrentShop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.called {
		t.Fatal("expected current shop service to be called")
	}
	if service.shopID != 2 {
		t.Fatalf("expected shop id 2, got %d", service.shopID)
	}

	var response struct {
		Success bool                `json:"success"`
		Data    CurrentShopResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.ID != 2 || response.Data.Name != "Bibas Kirana Pasal" || response.Data.Status != "active" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Data.Phone == nil || *response.Data.Phone != phone {
		t.Fatalf("expected phone %q, got %+v", phone, response.Data.Phone)
	}
}

func TestCurrentShopHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeCurrentShopService{})
	req := httptest.NewRequest(http.MethodGet, CurrentShopPath, nil)
	rec := httptest.NewRecorder()

	handler.CurrentShop(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCurrentShopHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeCurrentShopService{})
	req := httptest.NewRequest(http.MethodPost, CurrentShopPath, nil)
	rec := httptest.NewRecorder()

	handler.CurrentShop(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, rec.Header().Get("Allow"))
	}
}

func TestCurrentShopHandlerReturnsNotFound(t *testing.T) {
	service := &fakeCurrentShopService{err: ErrShopNotFound}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CurrentShopPath, nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CurrentShop(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestCurrentShopHandlerReturnsServerError(t *testing.T) {
	service := &fakeCurrentShopService{err: errors.New("database unavailable")}
	handler := NewHandler(service)

	ctx := contextx.WithShopID(context.Background(), "2")
	req := httptest.NewRequest(http.MethodGet, CurrentShopPath, nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CurrentShop(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

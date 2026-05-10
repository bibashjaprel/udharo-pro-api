package shop

import (
	"bytes"
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
	userID   int64
	role     string
	request  UpdateShopRequest
	called   bool
}

func (s *fakeCurrentShopService) CurrentShop(_ context.Context, shopID int64) (CurrentShopResponse, error) {
	s.called = true
	s.shopID = shopID
	return s.response, s.err
}

func (s *fakeCurrentShopService) UpdateShop(_ context.Context, userID int64, shopID int64, role string, req UpdateShopRequest) (CurrentShopResponse, error) {
	s.called = true
	s.userID = userID
	s.shopID = shopID
	s.role = role
	s.request = req
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

func TestUpdateShopHandlerUpdatesCurrentShop(t *testing.T) {
	phone := "9841000000"
	service := &fakeCurrentShopService{
		response: CurrentShopResponse{
			ID:     2,
			Name:   "Updated Shop",
			Phone:  &phone,
			Status: "active",
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")
	ctx = contextx.WithRole(ctx, "owner")

	body := []byte(`{"name":"Updated Shop","phone":"9841000000"}`)
	req := httptest.NewRequest(http.MethodPatch, CurrentShopPath, bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateShop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.called {
		t.Fatal("expected update shop service to be called")
	}
	if service.userID != 1 || service.shopID != 2 || service.role != "owner" {
		t.Fatalf("unexpected update args: user=%d shop=%d role=%q", service.userID, service.shopID, service.role)
	}
	if service.request.Name == nil || *service.request.Name != "Updated Shop" {
		t.Fatalf("expected name to be passed to service, got %+v", service.request.Name)
	}

	var response struct {
		Success bool                `json:"success"`
		Data    CurrentShopResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Name != "Updated Shop" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestUpdateShopHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeCurrentShopService{})
	req := httptest.NewRequest(http.MethodPatch, CurrentShopPath, bytes.NewReader([]byte(`{"name":"Updated Shop"}`)))
	rec := httptest.NewRecorder()

	handler.UpdateShop(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUpdateShopHandlerRejectsInvalidFields(t *testing.T) {
	handler := NewHandler(&fakeCurrentShopService{})

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")
	ctx = contextx.WithRole(ctx, "owner")

	req := httptest.NewRequest(http.MethodPatch, CurrentShopPath, bytes.NewReader([]byte(`{"unknown":"value"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateShop(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateShopHandlerRequiresOwner(t *testing.T) {
	service := &fakeCurrentShopService{err: ErrShopUpdateForbidden}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")
	ctx = contextx.WithRole(ctx, "staff")

	req := httptest.NewRequest(http.MethodPatch, CurrentShopPath, bytes.NewReader([]byte(`{"name":"Updated Shop"}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateShop(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUpdateShopHandlerRejectsInvalidProfile(t *testing.T) {
	service := &fakeCurrentShopService{err: ErrInvalidShopProfile}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")
	ctx = contextx.WithRole(ctx, "owner")

	req := httptest.NewRequest(http.MethodPatch, CurrentShopPath, bytes.NewReader([]byte(`{"name":" "}`))).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateShop(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

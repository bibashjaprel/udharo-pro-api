package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
)

func TestTenantMiddlewareRequiresShopID(t *testing.T) {
	handler := Tenant()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestTenantMiddlewareAllowsCurrentShop(t *testing.T) {
	handler := Tenant()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	ctx := contextx.WithShopID(t.Context(), "2")
	req := httptest.NewRequest(http.MethodGet, "/private", nil).WithContext(ctx)
	req.Header.Set(TenantHeader, "2")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestTenantMiddlewareAllowsMissingTenantHeader(t *testing.T) {
	handler := Tenant()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	ctx := contextx.WithShopID(t.Context(), "2")
	req := httptest.NewRequest(http.MethodGet, "/private", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestTenantMiddlewareRejectsAnotherShop(t *testing.T) {
	handler := Tenant()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	ctx := contextx.WithShopID(t.Context(), "2")
	req := httptest.NewRequest(http.MethodGet, "/private", nil).WithContext(ctx)
	req.Header.Set(TenantHeader, "3")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestTenantMiddlewareRejectsAnotherShopQuery(t *testing.T) {
	handler := Tenant()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	ctx := contextx.WithShopID(t.Context(), "2")
	req := httptest.NewRequest(http.MethodGet, "/private?shop_id=3", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

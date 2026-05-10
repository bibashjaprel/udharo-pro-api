package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
)

type fakeMeService struct {
	response CurrentUserResponse
	err      error
	userID   int64
	shopID   int64
	called   bool
}

func (s *fakeMeService) Signup(_ context.Context, _ SignupRequest) (SignupResponse, error) {
	return SignupResponse{}, nil
}

func (s *fakeMeService) Login(_ context.Context, _ LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func (s *fakeMeService) Logout(_ context.Context, _ string, _ int64, _ int64) error {
	return nil
}

func (s *fakeMeService) Me(_ context.Context, userID int64, shopID int64) (CurrentUserResponse, error) {
	s.called = true
	s.userID = userID
	s.shopID = shopID
	return s.response, s.err
}

func TestMeHandlerReturnsCurrentUserProfile(t *testing.T) {
	service := &fakeMeService{
		response: CurrentUserResponse{
			User: LoginUserInfo{
				ID:     1,
				Name:   "Bibas",
				Email:  "bibas@example.com",
				Phone:  "9841000000",
				Status: "active",
			},
			Shop: LoginShopInfo{
				ID:     2,
				Name:   "Bibas Kirana Pasal",
				Status: "active",
				Role:   "owner",
			},
		},
	}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.called {
		t.Fatal("expected me service to be called")
	}
	if service.userID != 1 || service.shopID != 2 {
		t.Fatalf("unexpected me args: user=%d shop=%d", service.userID, service.shopID)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("response must not expose password fields: %s", rec.Body.String())
	}

	var response struct {
		Success bool                `json:"success"`
		Data    CurrentUserResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.User.ID != 1 || response.Data.Shop.ID != 2 || response.Data.Shop.Role != "owner" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestMeHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeMeService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMeHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeMeService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, rec.Header().Get("Allow"))
	}
}

func TestMeHandlerRejectsMissingCurrentUser(t *testing.T) {
	service := &fakeMeService{err: ErrCurrentUserNotFound}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMeHandlerReturnsServerError(t *testing.T) {
	service := &fakeMeService{err: errors.New("database unavailable")}
	handler := NewHandler(service)

	ctx := contextx.WithUserID(context.Background(), "1")
	ctx = contextx.WithShopID(ctx, "2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

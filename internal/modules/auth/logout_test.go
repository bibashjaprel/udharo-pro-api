package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeLogoutService struct {
	err     error
	tokenID string
	userID  int64
	shopID  int64
	called  bool
}

func (s *fakeLogoutService) Signup(_ context.Context, _ SignupRequest) (SignupResponse, error) {
	return SignupResponse{}, nil
}

func (s *fakeLogoutService) Login(_ context.Context, _ LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func (s *fakeLogoutService) Logout(_ context.Context, tokenID string, userID int64, shopID int64) error {
	s.called = true
	s.tokenID = tokenID
	s.userID = userID
	s.shopID = shopID
	return s.err
}

func TestLogoutHandlerRevokesCurrentSession(t *testing.T) {
	service := &fakeLogoutService{}
	handler := NewHandler(service)

	ctx := context.WithValue(context.Background(), tokenIDContextKey, "token-id")
	ctx = context.WithValue(ctx, userIDContextKey, int64(1))
	ctx = context.WithValue(ctx, shopIDContextKey, int64(2))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.called {
		t.Fatal("expected logout service to be called")
	}
	if service.tokenID != "token-id" || service.userID != 1 || service.shopID != 2 {
		t.Fatalf("unexpected logout args: token=%q user=%d shop=%d", service.tokenID, service.userID, service.shopID)
	}

	var response LogoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message == "" {
		t.Fatal("expected success message")
	}
}

func TestLogoutHandlerRequiresAuthentication(t *testing.T) {
	handler := NewHandler(&fakeLogoutService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLogoutHandlerRejectsMissingSession(t *testing.T) {
	service := &fakeLogoutService{err: ErrSessionNotFound}
	handler := NewHandler(service)

	ctx := context.WithValue(context.Background(), tokenIDContextKey, "token-id")
	ctx = context.WithValue(ctx, userIDContextKey, int64(1))
	ctx = context.WithValue(ctx, shopIDContextKey, int64(2))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLogoutHandlerReturnsServerError(t *testing.T) {
	service := &fakeLogoutService{err: errors.New("database unavailable")}
	handler := NewHandler(service)

	ctx := context.WithValue(context.Background(), tokenIDContextKey, "token-id")
	ctx = context.WithValue(ctx, userIDContextKey, int64(1))
	ctx = context.WithValue(ctx, shopIDContextKey, int64(2))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

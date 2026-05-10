package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeLoginService struct {
	response LoginResponse
	err      error
	request  LoginRequest
	called   bool
}

func (s *fakeLoginService) Signup(_ context.Context, _ SignupRequest) (SignupResponse, error) {
	return SignupResponse{}, nil
}

func (s *fakeLoginService) Login(_ context.Context, req LoginRequest) (LoginResponse, error) {
	s.called = true
	s.request = req
	return s.response, s.err
}

func (s *fakeLoginService) Logout(_ context.Context, _ string, _ int64, _ int64) error {
	return nil
}

func TestLoginHandlerLogsInUser(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	service := &fakeLoginService{
		response: LoginResponse{
			AccessToken: "token",
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt,
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

	body := []byte(`{"identifier":"bibas@example.com","password":"StrongPassword123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.called {
		t.Fatal("expected login service to be called")
	}
	if service.request.Identifier != "bibas@example.com" {
		t.Fatalf("expected identifier to be passed to service, got %q", service.request.Identifier)
	}

	var response struct {
		Success bool          `json:"success"`
		Data    LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.AccessToken == "" || response.Data.User.ID != 1 || response.Data.Shop.ID != 2 || response.Data.Shop.Role != "owner" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestLoginHandlerRejectsInvalidCredentials(t *testing.T) {
	service := &fakeLoginService{err: ErrInvalidCredentials}
	handler := NewHandler(service)

	body := []byte(`{"identifier":"bibas@example.com","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLoginHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeLoginService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, rec.Header().Get("Allow"))
	}
}

func TestNormalizeLoginIdentifier(t *testing.T) {
	tests := []struct {
		name string
		req  LoginRequest
		want string
	}{
		{
			name: "identifier email",
			req:  LoginRequest{Identifier: " BIBAS@example.com "},
			want: "bibas@example.com",
		},
		{
			name: "email fallback",
			req:  LoginRequest{Email: " bibas@example.com "},
			want: "bibas@example.com",
		},
		{
			name: "email or phone fallback",
			req:  LoginRequest{EmailOrPhone: " 9841000000 "},
			want: "9841000000",
		},
		{
			name: "phone fallback",
			req:  LoginRequest{Phone: " 9841000000 "},
			want: "9841000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLoginIdentifier(tt.req)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestLoginRequiresIdentifierAndPassword(t *testing.T) {
	err := ErrInvalidCredentials
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeSignupService struct {
	response SignupResponse
	err      error
	request  SignupRequest
	called   bool
}

func (s *fakeSignupService) Signup(_ context.Context, req SignupRequest) (SignupResponse, error) {
	s.called = true
	s.request = req
	return s.response, s.err
}

func (s *fakeSignupService) Login(_ context.Context, _ LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func TestSignupHandlerCreatesSignup(t *testing.T) {
	service := &fakeSignupService{
		response: SignupResponse{
			UserID:         1,
			ShopID:         2,
			SubscriptionID: 3,
			Role:           "owner",
		},
	}
	handler := NewHandler(service)

	body := []byte(`{
		"name": "Bibas",
		"email": "bibas@example.com",
		"phone": "9841000000",
		"password": "StrongPassword123",
		"shop_name": "Bibas Kirana Pasal"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !service.called {
		t.Fatal("expected signup service to be called")
	}
	if service.request.Email != "bibas@example.com" {
		t.Fatalf("expected request email to be passed to service, got %q", service.request.Email)
	}

	var response SignupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Role != "owner" || response.UserID != 1 || response.ShopID != 2 || response.SubscriptionID != 3 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSignupHandlerRejectsDuplicateEmail(t *testing.T) {
	service := &fakeSignupService{err: ErrDuplicateEmail}
	handler := NewHandler(service)

	body := []byte(`{
		"name": "Bibas",
		"email": "bibas@example.com",
		"phone": "9841000000",
		"password": "StrongPassword123",
		"shop_name": "Bibas Kirana Pasal"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestSignupHandlerRejectsInvalidMethod(t *testing.T) {
	handler := NewHandler(&fakeSignupService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/signup", nil)
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestValidateSignupRequestRequiresFields(t *testing.T) {
	err := validateSignupRequest(SignupRequest{
		Name:     "Bibas",
		Email:    "bibas@example.com",
		Phone:    "9841000000",
		Password: "",
		ShopName: "Bibas Kirana Pasal",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

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
	response                   SignupResponse
	resendResponse             ResendEmailVerificationResponse
	verifyResponse             VerifyEmailResponse
	err                        error
	resendErr                  error
	verifyErr                  error
	request                    SignupRequest
	resendRequest              ResendEmailVerificationRequest
	verifyRequest              VerifyEmailRequest
	called                     bool
	resendEmailVerificationHit bool
	verifyEmailHit             bool
}

func (s *fakeSignupService) Signup(_ context.Context, req SignupRequest) (SignupResponse, error) {
	s.called = true
	s.request = req
	return s.response, s.err
}

func (s *fakeSignupService) ResendEmailVerification(_ context.Context, req ResendEmailVerificationRequest) (ResendEmailVerificationResponse, error) {
	s.resendEmailVerificationHit = true
	s.resendRequest = req
	return s.resendResponse, s.resendErr
}

func (s *fakeSignupService) VerifyEmail(_ context.Context, req VerifyEmailRequest) (VerifyEmailResponse, error) {
	s.verifyEmailHit = true
	s.verifyRequest = req
	return s.verifyResponse, s.verifyErr
}

func (s *fakeSignupService) Login(_ context.Context, _ LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func (s *fakeSignupService) Logout(_ context.Context, _ string, _ int64, _ int64) error {
	return nil
}

func (s *fakeSignupService) Me(_ context.Context, _ int64, _ int64) (CurrentUserResponse, error) {
	return CurrentUserResponse{}, nil
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

	var response struct {
		Success bool           `json:"success"`
		Data    SignupResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.Role != "owner" || response.Data.UserID != 1 || response.Data.ShopID != 2 || response.Data.SubscriptionID != 3 {
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

func TestSignupHandlerRejectsUnverifiedEmail(t *testing.T) {
	service := &fakeSignupService{err: ErrEmailVerificationRequired}
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

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestResendEmailVerificationHandler(t *testing.T) {
	service := &fakeSignupService{
		resendResponse: ResendEmailVerificationResponse{
			Email: "bibas@example.com",
		},
	}
	handler := NewHandler(service)

	body := []byte(`{"email":"bibas@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-verification/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ResendEmailVerification(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.resendEmailVerificationHit {
		t.Fatal("expected resend service to be called")
	}
	if service.resendRequest.Email != "bibas@example.com" {
		t.Fatalf("expected request email to be passed to service, got %q", service.resendRequest.Email)
	}
}

func TestVerifyEmailHandler(t *testing.T) {
	service := &fakeSignupService{}
	handler := NewHandler(service)

	body := []byte(`{"email":"bibas@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-verification/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.VerifyEmail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !service.verifyEmailHit {
		t.Fatal("expected verify service to be called")
	}
	if service.verifyRequest.Email != "bibas@example.com" || service.verifyRequest.Code != "123456" {
		t.Fatalf("unexpected verify request: %+v", service.verifyRequest)
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

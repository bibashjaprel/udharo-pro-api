package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type SignupService interface {
	Signup(ctx context.Context, req SignupRequest) (SignupResponse, error)
	ResendEmailVerification(ctx context.Context, req ResendEmailVerificationRequest) (ResendEmailVerificationResponse, error)
	VerifyEmail(ctx context.Context, req VerifyEmailRequest) (VerifyEmailResponse, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	Logout(ctx context.Context, tokenID string, userID int64, shopID int64) error
	Me(ctx context.Context, userID int64, shopID int64) (CurrentUserResponse, error)
}

type Handler struct {
	service SignupService
}

func NewHandler(service SignupService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	var req SignupRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.Signup(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "invalid request", "name, email, phone, password, and shop_name are required")
		case errors.Is(err, ErrEmailVerificationRequired):
			response.Error(w, http.StatusForbidden, "signup failed", "email verification required")
		case errors.Is(err, ErrDuplicateEmail):
			response.Error(w, http.StatusConflict, "signup failed", "email already exists")
		case errors.Is(err, ErrDuplicatePhone):
			response.Error(w, http.StatusConflict, "signup failed", "phone already exists")
		case errors.Is(err, ErrDuplicateSignup):
			response.Error(w, http.StatusConflict, "signup failed", "signup already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "signup failed", "signup failed")
		}
		return
	}

	response.Success(w, http.StatusCreated, "signup successful", res)
}

func (h *Handler) ResendEmailVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	var req ResendEmailVerificationRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.ResendEmailVerification(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "invalid request", "email is required")
		case errors.Is(err, ErrDuplicateEmail):
			response.Error(w, http.StatusConflict, "verification failed", "email already exists")
		case errors.Is(err, ErrEmailAlreadyVerified):
			response.Error(w, http.StatusConflict, "verification failed", "email already verified")
		default:
			response.Error(w, http.StatusInternalServerError, "verification failed", "verification email failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "verification email sent", res)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	var req VerifyEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.VerifyEmail(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "invalid request", "email and code are required")
		case errors.Is(err, ErrEmailVerificationNotFound), errors.Is(err, ErrInvalidVerificationCode):
			response.Error(w, http.StatusUnauthorized, "verification failed", "invalid verification code")
		case errors.Is(err, ErrEmailVerificationExpired):
			response.Error(w, http.StatusGone, "verification failed", "verification code expired")
		default:
			response.Error(w, http.StatusInternalServerError, "verification failed", "email verification failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "email verified", res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request", "invalid json body")
		return
	}

	res, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		default:
			response.Error(w, http.StatusInternalServerError, "login failed", "login failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "login successful", res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	tokenID, ok := contextx.GetTokenID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	userID, ok := contextx.GetUserIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	shopID, ok := contextx.GetShopIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	if err := h.service.Logout(r.Context(), tokenID, userID, shopID); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		default:
			response.Error(w, http.StatusInternalServerError, "logout failed", "logout failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "logged out successfully", LogoutResponse{Message: "logged out successfully"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	userID, ok := contextx.GetUserIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	shopID, ok := contextx.GetShopIDInt64(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	res, err := h.service.Me(r.Context(), userID, shopID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCurrentUserNotFound):
			response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		default:
			response.Error(w, http.StatusInternalServerError, "profile fetch failed", "profile fetch failed")
		}
		return
	}

	response.Success(w, http.StatusOK, "current user fetched successfully", res)
}

func decodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

type SignupService interface {
	Signup(ctx context.Context, req SignupRequest) (SignupResponse, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	Logout(ctx context.Context, tokenID string, userID int64, shopID int64) error
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

	userID, ok := contextx.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	shopID, ok := contextx.GetShopID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	parsedUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	parsedShopID, err := strconv.ParseInt(shopID, 10, 64)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	if err := h.service.Logout(r.Context(), tokenID, parsedUserID, parsedShopID); err != nil {
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

func decodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

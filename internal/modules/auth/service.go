package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const trialDuration = 14 * 24 * time.Hour

var (
	ErrInvalidInput       = errors.New("invalid signup input")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrDuplicatePhone     = errors.New("phone already exists")
	ErrDuplicateSignup    = errors.New("signup already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)

type Service struct {
	repository *Repository
	jwtSecret  string
}

func NewService(db *pgxpool.Pool, jwtSecret string) *Service {
	return &Service{
		repository: NewRepository(db),
		jwtSecret:  jwtSecret,
	}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (SignupResponse, error) {
	req = normalizeSignupRequest(req)
	if err := validateSignupRequest(req); err != nil {
		return SignupResponse{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repository.CreateSignup(ctx, req, string(passwordHash), trialDuration)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	identifier := normalizeLoginIdentifier(req)
	if identifier == "" || strings.TrimSpace(req.Password) == "" {
		return LoginResponse{}, ErrInvalidCredentials
	}

	session, passwordHash, err := s.repository.FindLoginSession(ctx, identifier)
	if err != nil {
		return LoginResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	tokenID, err := newTokenID()
	if err != nil {
		return LoginResponse{}, fmt.Errorf("create token id: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(accessTokenDuration)
	accessToken, err := SignAccessToken(session, tokenID, s.jwtSecret, now, expiresAt)
	if err != nil {
		return LoginResponse{}, err
	}

	if err := s.repository.CreateUserSession(ctx, session.User.ID, session.Shop.ID, tokenID, expiresAt); err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        session.User,
		Shop:        session.Shop,
	}, nil
}

func (s *Service) Logout(ctx context.Context, tokenID string, userID int64, shopID int64) error {
	return s.repository.RevokeSession(ctx, tokenID, userID, shopID)
}

func (s *Service) IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error) {
	return s.repository.IsSessionActive(ctx, tokenID, userID, shopID)
}

func normalizeSignupRequest(req SignupRequest) SignupRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	req.ShopName = strings.TrimSpace(req.ShopName)
	return req
}

func validateSignupRequest(req SignupRequest) error {
	if req.Name == "" || req.Email == "" || req.Phone == "" || req.Password == "" || req.ShopName == "" {
		return ErrInvalidInput
	}

	return nil
}

func normalizeLoginIdentifier(req LoginRequest) string {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.EmailOrPhone)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Phone)
	}
	if strings.Contains(identifier, "@") {
		identifier = strings.ToLower(identifier)
	}

	return identifier
}

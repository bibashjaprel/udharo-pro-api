package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const trialDuration = 14 * 24 * time.Hour
const emailVerificationDuration = 15 * time.Minute

var (
	ErrInvalidInput              = errors.New("invalid signup input")
	ErrDuplicateEmail            = errors.New("email already exists")
	ErrDuplicatePhone            = errors.New("phone already exists")
	ErrDuplicateSignup           = errors.New("signup already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrSessionNotFound           = errors.New("session not found")
	ErrCurrentUserNotFound       = errors.New("current user not found")
	ErrEmailVerificationRequired = errors.New("email verification required")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
	ErrEmailVerificationNotFound = errors.New("email verification not found")
	ErrEmailVerificationExpired  = errors.New("email verification expired")
	ErrInvalidVerificationCode   = errors.New("invalid verification code")
)

type EmailSender interface {
	SendVerificationCode(ctx context.Context, email string, code string) error
}

type LogEmailSender struct{}

func (LogEmailSender) SendVerificationCode(_ context.Context, email string, code string) error {
	log.Printf("email verification code for %s: %s", email, code)
	return nil
}

type ResendEmailSender struct {
	APIKey     string
	From       string
	HTTPClient *http.Client
}

func (s ResendEmailSender) SendVerificationCode(ctx context.Context, email string, code string) error {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]any{
		"from":    s.From,
		"to":      []string{email},
		"subject": "Verify your Udharo Pro email",
		"html": "<p>Your Udharo Pro verification code is <strong>" +
			html.EscapeString(code) +
			"</strong>.</p><p>It expires in 15 minutes.</p>",
		"text": "Your Udharo Pro verification code is " + code + ". It expires in 15 minutes.",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode resend email: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "udharo-pro-api/1.0")

	res, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send resend email: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("resend email failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

type Service struct {
	repository  *Repository
	jwtSecret   string
	emailSender EmailSender
}

func NewService(db *pgxpool.Pool, jwtSecret string) *Service {
	return NewServiceWithEmailSender(db, jwtSecret, LogEmailSender{})
}

func NewServiceWithEmailSender(db *pgxpool.Pool, jwtSecret string, emailSender EmailSender) *Service {
	if emailSender == nil {
		emailSender = LogEmailSender{}
	}

	return &Service{
		repository:  NewRepository(db),
		jwtSecret:   jwtSecret,
		emailSender: emailSender,
	}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (SignupResponse, error) {
	req = normalizeSignupRequest(req)
	if err := validateSignupRequest(req); err != nil {
		return SignupResponse{}, err
	}
	if err := s.repository.RequireVerifiedEmail(ctx, req.Email); err != nil {
		return SignupResponse{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return SignupResponse{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repository.CreateSignup(ctx, req, string(passwordHash), trialDuration)
}

func (s *Service) ResendEmailVerification(ctx context.Context, req ResendEmailVerificationRequest) (ResendEmailVerificationResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return ResendEmailVerificationResponse{}, ErrInvalidInput
	}

	exists, err := s.repository.UserEmailExists(ctx, email)
	if err != nil {
		return ResendEmailVerificationResponse{}, err
	}
	if exists {
		return ResendEmailVerificationResponse{}, ErrDuplicateEmail
	}

	verified, err := s.repository.IsEmailVerified(ctx, email)
	if err != nil {
		return ResendEmailVerificationResponse{}, err
	}
	if verified {
		return ResendEmailVerificationResponse{}, ErrEmailAlreadyVerified
	}

	code, err := newVerificationCode()
	if err != nil {
		return ResendEmailVerificationResponse{}, fmt.Errorf("create verification code: %w", err)
	}
	expiresAt := time.Now().UTC().Add(emailVerificationDuration)
	if err := s.repository.SaveEmailVerification(ctx, email, hashVerificationCode(code), expiresAt); err != nil {
		return ResendEmailVerificationResponse{}, err
	}
	if err := s.emailSender.SendVerificationCode(ctx, email, code); err != nil {
		return ResendEmailVerificationResponse{}, fmt.Errorf("send verification email: %w", err)
	}

	return ResendEmailVerificationResponse{Email: email, ExpiresAt: expiresAt}, nil
}

func (s *Service) VerifyEmail(ctx context.Context, req VerifyEmailRequest) (VerifyEmailResponse, error) {
	email := normalizeEmail(req.Email)
	code := strings.TrimSpace(req.Code)
	if email == "" || code == "" {
		return VerifyEmailResponse{}, ErrInvalidInput
	}

	verification, err := s.repository.FindEmailVerification(ctx, email)
	if err != nil {
		return VerifyEmailResponse{}, err
	}
	if time.Now().UTC().After(verification.ExpiresAt) {
		return VerifyEmailResponse{}, ErrEmailVerificationExpired
	}
	if subtle.ConstantTimeCompare([]byte(verification.CodeHash), []byte(hashVerificationCode(code))) != 1 {
		return VerifyEmailResponse{}, ErrInvalidVerificationCode
	}

	verifiedAt, err := s.repository.MarkEmailVerified(ctx, email)
	if err != nil {
		return VerifyEmailResponse{}, err
	}

	return VerifyEmailResponse{Email: email, VerifiedAt: verifiedAt}, nil
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

func (s *Service) Me(ctx context.Context, userID int64, shopID int64) (CurrentUserResponse, error) {
	return s.repository.FindCurrentUser(ctx, userID, shopID)
}

func (s *Service) IsSessionActive(ctx context.Context, tokenID string, userID int64, shopID int64) (bool, error) {
	return s.repository.IsSessionActive(ctx, tokenID, userID, shopID)
}

func normalizeSignupRequest(req SignupRequest) SignupRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = normalizeEmail(req.Email)
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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

func newVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashVerificationCode(code string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(hash[:])
}

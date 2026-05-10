package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenDuration = 24 * time.Hour

type AccessTokenClaims struct {
	UserID int64  `json:"user_id"`
	ShopID int64  `json:"shop_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func SignAccessToken(session loginSession, tokenID string, jwtSecret string, issuedAt time.Time, expiresAt time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessTokenClaims{
		UserID: session.User.ID,
		ShopID: session.Shop.ID,
		Role:   session.Shop.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", session.User.ID),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	accessToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return accessToken, nil
}

func ParseAccessToken(tokenString string, jwtSecret string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("invalid signing method")
			}

			return []byte(jwtSecret), nil
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.ID == "" || claims.UserID == 0 || claims.ShopID == 0 {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func newTokenID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}

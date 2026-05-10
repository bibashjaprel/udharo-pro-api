package auth

import "time"

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	ShopName string `json:"shop_name"`
}

type SignupResponse struct {
	UserID         int64  `json:"user_id"`
	ShopID         int64  `json:"shop_id"`
	SubscriptionID int64  `json:"subscription_id"`
	Role           string `json:"role"`
}

type LoginRequest struct {
	Identifier   string `json:"identifier"`
	EmailOrPhone string `json:"email_or_phone"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
}

type LoginResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
	User        LoginUserInfo `json:"user"`
	Shop        LoginShopInfo `json:"shop"`
}

type LoginUserInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

type LoginShopInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Role   string `json:"role"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type CurrentUserResponse struct {
	User LoginUserInfo `json:"user"`
	Shop LoginShopInfo `json:"shop"`
}

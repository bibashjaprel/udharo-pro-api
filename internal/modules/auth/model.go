package auth

import "time"

type loginSession struct {
	User LoginUserInfo
	Shop LoginShopInfo
}

type emailVerification struct {
	CodeHash  string
	ExpiresAt time.Time
}

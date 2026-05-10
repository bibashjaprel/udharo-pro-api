package contextx

import (
	"context"
	"strconv"
)

type contextKey string

const (
	userIDKey  contextKey = "user_id"
	shopIDKey  contextKey = "shop_id"
	roleKey    contextKey = "role"
	tokenIDKey contextKey = "token_id"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func WithShopID(ctx context.Context, shopID string) context.Context {
	return context.WithValue(ctx, shopIDKey, shopID)
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

func GetShopID(ctx context.Context) (string, bool) {
	shopID, ok := ctx.Value(shopIDKey).(string)
	return shopID, ok
}

func GetShopIDInt64(ctx context.Context) (int64, bool) {
	shopID, ok := GetShopID(ctx)
	if !ok {
		return 0, false
	}

	parsedShopID, err := strconv.ParseInt(shopID, 10, 64)
	if err != nil {
		return 0, false
	}

	return parsedShopID, true
}

func GetRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}

func WithTokenID(ctx context.Context, tokenID string) context.Context {
	return context.WithValue(ctx, tokenIDKey, tokenID)
}

func GetTokenID(ctx context.Context) (string, bool) {
	tokenID, ok := ctx.Value(tokenIDKey).(string)
	return tokenID, ok
}

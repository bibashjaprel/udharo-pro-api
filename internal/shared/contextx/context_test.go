package contextx

import "testing"

func TestGetUserIDInt64(t *testing.T) {
	ctx := WithUserID(t.Context(), "7")

	userID, ok := GetUserIDInt64(ctx)
	if !ok || userID != 7 {
		t.Fatalf("expected user id 7, got %d, %v", userID, ok)
	}
}

func TestGetUserIDInt64RejectsInvalidValue(t *testing.T) {
	ctx := WithUserID(t.Context(), "not-a-number")

	userID, ok := GetUserIDInt64(ctx)
	if ok || userID != 0 {
		t.Fatalf("expected invalid user id, got %d, %v", userID, ok)
	}
}

func TestGetShopIDInt64(t *testing.T) {
	ctx := WithShopID(t.Context(), "42")

	shopID, ok := GetShopIDInt64(ctx)
	if !ok || shopID != 42 {
		t.Fatalf("expected shop id 42, got %d, %v", shopID, ok)
	}
}

func TestGetShopIDInt64RejectsInvalidValue(t *testing.T) {
	ctx := WithShopID(t.Context(), "not-a-number")

	shopID, ok := GetShopIDInt64(ctx)
	if ok || shopID != 0 {
		t.Fatalf("expected invalid shop id, got %d, %v", shopID, ok)
	}
}

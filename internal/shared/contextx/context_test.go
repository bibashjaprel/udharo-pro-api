package contextx

import "testing"

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

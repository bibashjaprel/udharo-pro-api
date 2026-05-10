package shop

import (
	"errors"
	"testing"
)

func TestNormalizeUpdateShopRequestRequiresAField(t *testing.T) {
	_, err := normalizeUpdateShopRequest(UpdateShopRequest{})
	if !errors.Is(err, ErrInvalidShopProfile) {
		t.Fatalf("expected ErrInvalidShopProfile, got %v", err)
	}
}

func TestNormalizeUpdateShopRequestRejectsBlankName(t *testing.T) {
	name := " "
	_, err := normalizeUpdateShopRequest(UpdateShopRequest{Name: &name})
	if !errors.Is(err, ErrInvalidShopProfile) {
		t.Fatalf("expected ErrInvalidShopProfile, got %v", err)
	}
}

func TestNormalizeUpdateShopRequestTrimsAndClearsOptionalFields(t *testing.T) {
	name := " Updated Shop "
	phone := " "
	address := " Kathmandu "

	fields, err := normalizeUpdateShopRequest(UpdateShopRequest{
		Name:    &name,
		Phone:   &phone,
		Address: &address,
	})
	if err != nil {
		t.Fatalf("normalize update shop request: %v", err)
	}
	if !fields.NameSet || fields.Name != "Updated Shop" {
		t.Fatalf("expected trimmed name, got %+v", fields)
	}
	if !fields.PhoneSet || fields.Phone != nil {
		t.Fatalf("expected phone to be cleared, got %+v", fields.Phone)
	}
	if !fields.AddressSet || fields.Address == nil || *fields.Address != "Kathmandu" {
		t.Fatalf("expected trimmed address, got %+v", fields.Address)
	}
}

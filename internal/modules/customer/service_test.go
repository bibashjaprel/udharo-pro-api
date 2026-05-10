package customer

import (
	"errors"
	"testing"
)

func TestNormalizeCreateCustomerRequest(t *testing.T) {
	address := " Kathmandu "
	notes := " "

	req := normalizeCreateCustomerRequest(CreateCustomerRequest{
		Name:    " Ram Bahadur ",
		Phone:   " 9841000000 ",
		Address: &address,
		Notes:   &notes,
	})

	if req.Name != "Ram Bahadur" || req.Phone != "9841000000" {
		t.Fatalf("expected trimmed name and phone, got %+v", req)
	}
	if req.Address == nil || *req.Address != "Kathmandu" {
		t.Fatalf("expected trimmed address, got %+v", req.Address)
	}
	if req.Notes != nil {
		t.Fatalf("expected blank notes to be nil, got %+v", req.Notes)
	}
}

func TestValidateCreateCustomerRequestRequiresNameAndPhone(t *testing.T) {
	err := validateCreateCustomerRequest(CreateCustomerRequest{Name: "Ram"})
	if !errors.Is(err, ErrInvalidCustomer) {
		t.Fatalf("expected ErrInvalidCustomer, got %v", err)
	}
}

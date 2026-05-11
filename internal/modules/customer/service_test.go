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

func TestNormalizeListCustomersRequestDefaultsPagination(t *testing.T) {
	req := normalizeListCustomersRequest(ListCustomersRequest{Search: " ram "})

	if req.Page != 1 || req.Limit != 20 {
		t.Fatalf("expected default pagination, got %+v", req)
	}
	if req.Search != "ram" {
		t.Fatalf("expected trimmed search, got %q", req.Search)
	}
}

func TestValidateListCustomersRequestRejectsInvalidPagination(t *testing.T) {
	tests := []ListCustomersRequest{
		{Page: 0, Limit: 20},
		{Page: 1, Limit: 0},
		{Page: 1, Limit: 101},
	}

	for _, tt := range tests {
		if err := validateListCustomersRequest(tt); !errors.Is(err, ErrInvalidPagination) {
			t.Fatalf("expected ErrInvalidPagination for %+v, got %v", tt, err)
		}
	}
}

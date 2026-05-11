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

func TestNormalizeUpdateCustomerRequest(t *testing.T) {
	name := " Ram Bahadur "
	phone := " 9841000001 "
	address := " "
	notes := " Updated notes "

	fields, err := normalizeUpdateCustomerRequest(UpdateCustomerRequest{
		Name:    &name,
		Phone:   &phone,
		Address: &address,
		Notes:   &notes,
	})
	if err != nil {
		t.Fatalf("normalize update request: %v", err)
	}

	if !fields.NameSet || fields.Name != "Ram Bahadur" {
		t.Fatalf("expected trimmed name, got %+v", fields)
	}
	if !fields.PhoneSet || fields.Phone != "9841000001" {
		t.Fatalf("expected trimmed phone, got %+v", fields)
	}
	if !fields.AddressSet || fields.Address != nil {
		t.Fatalf("expected blank address to be nil, got %+v", fields.Address)
	}
	if !fields.NotesSet || fields.Notes == nil || *fields.Notes != "Updated notes" {
		t.Fatalf("expected trimmed notes, got %+v", fields.Notes)
	}
}

func TestNormalizeUpdateCustomerRequestRejectsInvalidRequest(t *testing.T) {
	blank := " "
	tests := []UpdateCustomerRequest{
		{},
		{Name: &blank},
		{Phone: &blank},
	}

	for _, tt := range tests {
		if _, err := normalizeUpdateCustomerRequest(tt); !errors.Is(err, ErrInvalidCustomer) {
			t.Fatalf("expected ErrInvalidCustomer for %+v, got %v", tt, err)
		}
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

func TestGetCustomerRejectsInvalidID(t *testing.T) {
	service := &Service{}

	_, err := service.GetCustomer(nil, 1, 0)
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdateCustomerRejectsInvalidID(t *testing.T) {
	service := &Service{}

	_, err := service.UpdateCustomer(nil, 1, 1, 0, UpdateCustomerRequest{})
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

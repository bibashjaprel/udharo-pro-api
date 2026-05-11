package ledger

import (
	"errors"
	"testing"
)

func TestNormalizeCreateCreditEntryRequest(t *testing.T) {
	note := " Rice, oil, sugar "

	fields, err := normalizeCreateCreditEntryRequest(CreateCreditEntryRequest{
		Amount:          1500,
		Note:            &note,
		TransactionDate: "2026-05-09",
	})
	if err != nil {
		t.Fatalf("normalize credit entry: %v", err)
	}

	if fields.Amount != 1500 {
		t.Fatalf("expected amount 1500, got %v", fields.Amount)
	}
	if fields.Note == nil || *fields.Note != "Rice, oil, sugar" {
		t.Fatalf("expected trimmed note, got %+v", fields.Note)
	}
	if got := fields.TransactionDate.Format("2006-01-02"); got != "2026-05-09" {
		t.Fatalf("expected transaction date 2026-05-09, got %s", got)
	}
}

func TestNormalizeCreateCreditEntryRequestRejectsInvalidInput(t *testing.T) {
	tests := []CreateCreditEntryRequest{
		{Amount: 0},
		{Amount: -1},
		{Amount: 1500, TransactionDate: "05-09-2026"},
	}

	for _, tt := range tests {
		if _, err := normalizeCreateCreditEntryRequest(tt); !errors.Is(err, ErrInvalidCreditEntry) {
			t.Fatalf("expected ErrInvalidCreditEntry for %+v, got %v", tt, err)
		}
	}
}

func TestNormalizeListLedgerEntriesRequestDefaultsPagination(t *testing.T) {
	req := normalizeListLedgerEntriesRequest(ListLedgerEntriesRequest{})

	if req.Page != 1 || req.Limit != 20 {
		t.Fatalf("expected default pagination, got %+v", req)
	}
}

func TestValidateListLedgerEntriesRequestRejectsInvalidPagination(t *testing.T) {
	tests := []ListLedgerEntriesRequest{
		{Page: 0, Limit: 20},
		{Page: 1, Limit: 0},
		{Page: 1, Limit: 101},
	}

	for _, tt := range tests {
		if err := validateListLedgerEntriesRequest(tt); !errors.Is(err, ErrInvalidPagination) {
			t.Fatalf("expected ErrInvalidPagination for %+v, got %v", tt, err)
		}
	}
}

func TestCreateCreditEntryRejectsInvalidCustomerID(t *testing.T) {
	service := &Service{}

	_, err := service.CreateCreditEntry(nil, 1, 1, 0, CreateCreditEntryRequest{Amount: 1500})
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestListCustomerLedgerRejectsInvalidCustomerID(t *testing.T) {
	service := &Service{}

	_, err := service.ListCustomerLedger(nil, 1, 0, ListLedgerEntriesRequest{})
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

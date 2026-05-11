package ledger

import "net/http"

func CreditRoute(handler *Handler) http.Handler {
	return http.HandlerFunc(handler.CreateCreditEntry)
}

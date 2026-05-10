package errors

import stderrors "errors"

var (
	ErrInvalidRequest = stderrors.New("invalid request")
	ErrUnauthorized   = stderrors.New("unauthorized")
	ErrForbidden      = stderrors.New("forbidden")
	ErrNotFound       = stderrors.New("not found")
	ErrConflict       = stderrors.New("conflict")
)

package domain

import "errors"

var (
	ErrContextNotFound = errors.New("user context not found")
	ErrContextTooLong  = errors.New("context must be at most 4000 characters")
)

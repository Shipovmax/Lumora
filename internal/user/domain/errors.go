package domain

import "errors"

var (
	ErrProfileNotFound = errors.New("user profile not found")
	ErrNameTooLong     = errors.New("name must be at most 200 characters")
)

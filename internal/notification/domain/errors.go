package domain

import "errors"

var (
	ErrTokenRequired = errors.New("device token is required")
	ErrInvalidToken  = errors.New("device token is invalid or unregistered")
)

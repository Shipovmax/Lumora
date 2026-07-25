package domain

import "errors"

var (
	ErrEmailTaken          = errors.New("email is already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserNotFound        = errors.New("user not found")
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid, expired or revoked")
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
)

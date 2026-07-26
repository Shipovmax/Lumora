package domain

import "errors"

var (
	ErrSourceNotFound       = errors.New("source not found")
	ErrInvalidType          = errors.New("source type must be one of: rss, youtube, telegram")
	ErrNameRequired         = errors.New("source name is required")
	ErrURLRequired          = errors.New("source url is required")
	ErrUnsupportedURLScheme = errors.New("source url must use http or https")
)

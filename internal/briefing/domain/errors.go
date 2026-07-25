package domain

import "errors"

var (
	ErrNoRelevantEvents = errors.New("no relevant events for briefing")
	ErrInvalidType      = errors.New("briefing type must be one of: morning, evening")
)

// Package domain содержит сущности и порты (интерфейсы) домена briefing.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

type Type string

const (
	TypeMorning Type = "morning"
	TypeEvening Type = "evening"
)

func (t Type) Valid() bool {
	switch t {
	case TypeMorning, TypeEvening:
		return true
	default:
		return false
	}
}

// CandidateEvent — событие-кандидат в брифинг: минимальные данные, нужные
// для отбора и упорядочивания, до генерации объяснения.
type CandidateEvent struct {
	ID         string
	Topic      string
	Title      string
	Importance int
}

// BriefingEvent — событие в составе собранного брифинга вместе с его
// персонализированным объяснением (Этап 8), в порядке важности.
type BriefingEvent struct {
	EventID            string
	Topic              string
	Title              string
	Importance         int
	Rank               int
	WhatHappened       string
	WhyItHappened      string
	WhatChanged        string
	WhatItMeansForUser string
}

type Briefing struct {
	ID          string
	UserID      string
	Type        Type
	GeneratedAt time.Time
	Events      []BriefingEvent
}

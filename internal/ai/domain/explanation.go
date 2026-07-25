// Package domain содержит сущности и порты (интерфейсы) домена ai.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

// Explanation — персонализированное AI-объяснение события: четыре блока
// (что произошло/почему/что изменилось/что это значит для пользователя),
// сгенерированные с учётом контекста конкретного пользователя (Этап 4).
// Одна пара (event_id, user_id) — одно объяснение: "что это значит лично
// для вас" зависит от того, для кого генерируется, а не только от события.
type Explanation struct {
	ID                 string
	EventID            string
	UserID             string
	WhatHappened       string
	WhyItHappened      string
	WhatChanged        string
	WhatItMeansForUser string
	Model              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

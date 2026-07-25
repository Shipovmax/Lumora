package domain

import "context"

// ProfileUpdate — набор редактируемых полей профиля (полная замена, semantics PUT).
type ProfileUpdate struct {
	Name       string
	Country    string
	Language   string
	Profession string
	Interests  []string
	Topics     []string
}

// Repository — порт доступа к данным домена user. Реализуется в
// internal/user/repository поверх Postgres/sqlc.
type Repository interface {
	// GetProfile возвращает профиль пользователя, создавая пустой (default) при первом обращении.
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpdateProfile(ctx context.Context, userID string, update ProfileUpdate) (Profile, error)
}

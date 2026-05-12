package domain

import "time"

// User представляет зарегистрированного пользователя системы лояльности.
type User struct {
	// ID содержит внутренний идентификатор пользователя.
	ID int64

	// Login содержит уникальный логин пользователя.
	Login string

	// PasswordHash содержит bcrypt-хеш пароля пользователя.
	PasswordHash string

	// CreatedAt содержит время регистрации пользователя.
	CreatedAt time.Time
}

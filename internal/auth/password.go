// Package auth содержит функции аутентификации и авторизации пользователей.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword возвращает bcrypt-хеш переданного пароля.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword сообщает, соответствует ли пароль сохраненному bcrypt-хешу.
func CheckPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

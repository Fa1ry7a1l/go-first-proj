package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Ошибки проверки токена авторизации.
var (
	// ErrInvalidToken означает, что токен отсутствует, поврежден или имеет неверную подпись.
	ErrInvalidToken = errors.New("invalid token")
)

// TokenManager выпускает и проверяет подписанные токены пользователя.
type TokenManager struct {
	secret []byte
}

// NewTokenManager создает менеджер токенов с указанным секретом подписи.
// Если секрет пустой, используется случайный секрет на время жизни процесса.
func NewTokenManager(secret string) *TokenManager {
	if secret == "" {
		return &TokenManager{
			secret: randomSecret(),
		}
	}
	return &TokenManager{
		secret: []byte(secret),
	}
}

// Issue создает подписанный токен для пользователя.
func (m *TokenManager) Issue(userID int64) string {
	payload := strconv.FormatInt(userID, 10)
	signature := m.sign(payload)
	token := payload + "." + signature
	return base64.RawURLEncoding.EncodeToString([]byte(token))
}

// Verify проверяет токен и возвращает идентификатор пользователя.
func (m *TokenManager) Verify(token string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidToken
	}

	parts := strings.Split(string(raw), ".")
	if len(parts) != 2 {
		return 0, ErrInvalidToken
	}

	payload := parts[0]
	signature := parts[1]
	if !hmac.Equal([]byte(signature), []byte(m.sign(payload))) {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(payload, 10, 64)
	if err != nil || userID <= 0 {
		return 0, ErrInvalidToken
	}
	return userID, nil
}

func (m *TokenManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func randomSecret() []byte {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return []byte(strconv.FormatInt(timeNowUnixNano(), 10))
	}
	return secret
}

var timeNowUnixNano = func() int64 {
	return time.Now().UnixNano()
}

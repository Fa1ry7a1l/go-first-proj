package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage"
)

// UserService выполняет бизнес-операции регистрации и входа пользователей.
type UserService struct {
	users storage.UserStorage
}

// NewUserService создает сервис пользователей поверх переданного хранилища.
func NewUserService(users storage.UserStorage) *UserService {
	return &UserService{
		users: users,
	}
}

// Register создает нового пользователя с bcrypt-хешем пароля.
func (s *UserService) Register(ctx context.Context, login string, password string) (domain.User, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return domain.User{}, domain.ErrUserInvalidCredentialsFormat
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}

	user, err := s.users.CreateUser(ctx, domain.User{
		Login:        login,
		PasswordHash: passwordHash,
	})
	if errors.Is(err, domain.ErrUserAlreadyExists) {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

// Login проверяет логин и пароль пользователя.
func (s *UserService) Login(ctx context.Context, login string, password string) (domain.User, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return domain.User{}, domain.ErrUserInvalidCredentialsFormat
	}

	user, err := s.users.GetUserByLogin(ctx, login)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return domain.User{}, domain.ErrInvalidCredentials
	}

	return user, nil
}

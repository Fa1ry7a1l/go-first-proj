package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

func TestUserServiceRegister(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserStorage{}
	service := NewUserService(users)

	user, err := service.Register(ctx, " user ", "secret")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.Login != "user" {
		t.Fatalf("Login = %q, want user", user.Login)
	}
	if user.PasswordHash == "secret" {
		t.Fatal("password stored without hashing")
	}
	if !auth.CheckPassword(user.PasswordHash, "secret") {
		t.Fatal("stored hash does not match password")
	}
}

func TestUserServiceRegisterRejectsEmptyFields(t *testing.T) {
	service := NewUserService(&fakeUserStorage{})

	_, err := service.Register(context.Background(), "", "secret")
	if !errors.Is(err, domain.ErrUserInvalidCredentialsFormat) {
		t.Fatalf("Register error = %v, want %v", err, domain.ErrUserInvalidCredentialsFormat)
	}
}

func TestUserServiceLogin(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	service := NewUserService(&fakeUserStorage{
		user: &domain.User{ID: 7, Login: "user", PasswordHash: hash},
	})

	user, err := service.Login(context.Background(), "user", "secret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if user.ID != 7 {
		t.Fatalf("ID = %d, want 7", user.ID)
	}
}

func TestUserServiceLoginRejectsInvalidPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	service := NewUserService(&fakeUserStorage{
		user: &domain.User{ID: 7, Login: "user", PasswordHash: hash},
	})

	_, err = service.Login(context.Background(), "user", "wrong")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want %v", err, domain.ErrInvalidCredentials)
	}
}

type fakeUserStorage struct {
	user      *domain.User
	createErr error
}

func (s *fakeUserStorage) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if s.createErr != nil {
		return domain.User{}, s.createErr
	}
	user.ID = 1
	s.user = &user
	return user, nil
}

func (s *fakeUserStorage) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	if s.user == nil || s.user.Login != login {
		return domain.User{}, domain.ErrUserNotFound
	}
	return *s.user, nil
}

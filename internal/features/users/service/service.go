package users_service

import (
	"context"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

type PasswordHasher interface {
	Hash(password string) ([]byte, error)
	Compare(hash []byte, password string) error
}

type UsersService struct {
	usersRepository UsersRepository
	passwordHasher  PasswordHasher
}

type UsersRepository interface {
	CreateUser(
		ctx context.Context,
		login string,
		passwordHash []byte,
	) (domain.User, error)
}

func NewUsersService(
	usersRepository UsersRepository,
	passwordHasher PasswordHasher,
) *UsersService {
	return &UsersService{
		usersRepository: usersRepository,
		passwordHasher:  passwordHasher,
	}
}

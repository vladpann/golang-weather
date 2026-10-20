package users_service

import (
	"errors"
	"fmt"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

func NewPasswordHasher() *Hasher {
	return &Hasher{}
}

func (p *Hasher) Hash(password string) ([]byte, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return passwordHash, nil
}

func (p *Hasher) Compare(hash []byte, password string) error {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return core_errors.ErrInvalidCredentials
		}
		return fmt.Errorf("compare password: %w", err)
	}

	return nil
}

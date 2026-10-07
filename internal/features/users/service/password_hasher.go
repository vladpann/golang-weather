package users_service

import "golang.org/x/crypto/bcrypt"

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
		return err
	}

	return nil
}

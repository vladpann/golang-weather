package users_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (r *UsersRepository) CreateUser(
	ctx context.Context,
	login string,
	passwordHash []byte,
) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO weather.users (login, password)
	VALUES ($1, $2)
	RETURNING id, login;
	`

	row := r.pool.QueryRow(ctx, query, login, passwordHash)

	var userModel UserModel
	err := row.Scan(
		&userModel.ID,
		&userModel.Login,
	)

	if err != nil {
		return domain.User{}, fmt.Errorf("scan error: %w", err)
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, nil
}

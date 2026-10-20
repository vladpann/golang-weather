package users_repository_postgres

import (
	"context"
	"errors"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_postgres_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool"
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
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain.User{}, core_errors.ErrLoginAlreadyExists
		}
		return domain.User{}, err
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, nil
}

package users_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_postgres_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool"
)

func (r *UsersRepository) GetUser(
	ctx context.Context,
	login string,
) (user domain.User, passwordHash []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, login, password
		FROM weather.users
		WHERE login = $1
	`

	row := r.pool.QueryRow(ctx, query, login)

	var userModel UserModel
	var password []byte

	err = row.Scan(
		&userModel.ID,
		&userModel.Login,
		&password,
	)

	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.User{}, nil, fmt.Errorf("user with login='%s': %w",
				login,
				core_errors.ErrInvalidCredentials,
			)
		}

		return domain.User{}, nil, fmt.Errorf("scan user: %w", err)
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, password, nil
}

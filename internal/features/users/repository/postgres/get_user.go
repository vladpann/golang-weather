package users_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
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
		return domain.User{}, nil, fmt.Errorf("scan user: %w", err)
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, password, nil
}

package users_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (r *UsersRepository) GetUserByID(
	ctx context.Context,
	userID int64,
) (domain.User, error) {
	const query = `
		SELECT id, login
		FROM weather.users
		WHERE id = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Login)
	if err != nil {
		return domain.User{}, fmt.Errorf("query row: %w", err)
	}

	return user, nil
}

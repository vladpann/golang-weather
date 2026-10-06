package users_repository_postgres

import (
	core_repository_postgres_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool"
)

type UsersRepository struct {
	pool core_repository_postgres_pool.Pool
}

func NewUsersRepository(
	pool core_repository_postgres_pool.Pool,
) *UsersRepository {
	return &UsersRepository{
		pool: pool,
	}
}

package sessions_repository_postgres

import (
	core_repository_postgres_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool"
)

type SessionsRepository struct {
	pool core_repository_postgres_pool.Pool
}

func NewSessionsRepository(
	pool core_repository_postgres_pool.Pool,
) *SessionsRepository {
	return &SessionsRepository{
		pool: pool,
	}
}

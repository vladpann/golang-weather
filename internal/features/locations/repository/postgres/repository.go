package locations_repository_postgres

import (
	core_repository_postgres_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool"
)

type LocationsRepository struct {
	pool core_repository_postgres_pool.Pool
}

func NewLocationsRepository(
	pool core_repository_postgres_pool.Pool,
) *LocationsRepository {
	return &LocationsRepository{
		pool: pool,
	}
}

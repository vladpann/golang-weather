package locations_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_postgres_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool"
)

func (r *LocationsRepository) CreateLocation(
	ctx context.Context,
	location domain.Location,
) (domain.Location, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO weather.locations (user_id, name, latitude, longitude)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, latitude, longitude
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		location.UserID,
		location.Name,
		location.Latitude,
		location.Longitude,
	)

	var locationModel LocationModel
	err := row.Scan(
		&locationModel.ID,
		&locationModel.UserID,
		&locationModel.Name,
		&locationModel.Latitude,
		&locationModel.Longitude,
	)

	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain.Location{}, fmt.Errorf(
				"location with coordinates (%v, %v) for user %d: %w",
				location.Latitude,
				location.Longitude,
				location.UserID,
				core_errors.ErrLocationAlreadyExists,
			)
		}

		return domain.Location{}, fmt.Errorf("scan location: %w", err)
	}

	locationDomain := locationDomainFromModel(locationModel)

	return locationDomain, nil
}

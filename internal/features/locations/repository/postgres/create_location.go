package locations_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
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
	if err := row.Scan(
		&locationModel.ID,
		&locationModel.UserID,
		&locationModel.Name,
		&locationModel.Latitude,
		&locationModel.Longitude,
	); err != nil {
		return domain.Location{}, fmt.Errorf("scan location: %w", err)
	}

	locationDomain := locationDomainFromModel(locationModel)

	return locationDomain, nil
}

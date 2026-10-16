package locations_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (r *LocationsRepository) GetLocation(
	ctx context.Context,
	location domain.Location,
) (domain.Location, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, user_id, name, latitude, longitude
		FROM weather.locations
		WHERE id = $1 AND user_id = $2
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		location.ID,
		location.UserID,
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

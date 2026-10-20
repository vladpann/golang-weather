package locations_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (r *LocationsRepository) GetLocations(
	ctx context.Context,
	userID int64,
	limit *int,
	offset *int,
) ([]domain.Location, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT id, user_id, name, latitude, longitude
	FROM weather.locations
	WHERE user_id = $1
	ORDER BY id ASC
	LIMIT $2
	OFFSET $3;
	`

	rows, err := r.pool.Query(
		ctx,
		query,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("select locations: %w", err)
	}
	defer rows.Close()

	var locationModels []LocationModel
	for rows.Next() {
		var locationModel LocationModel

		err := rows.Scan(
			&locationModel.ID,
			&locationModel.UserID,
			&locationModel.Name,
			&locationModel.Latitude,
			&locationModel.Longitude,
		)
		if err != nil {
			return nil, fmt.Errorf("scan locations: %w", err)
		}

		locationModels = append(locationModels, locationModel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	locationDomains := locationDomainsFromModels(locationModels)

	return locationDomains, nil
}

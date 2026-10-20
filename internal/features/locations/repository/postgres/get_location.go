package locations_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func (r *LocationsRepository) GetLocation(
	ctx context.Context,
	id int64,
	userID int64,
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
		id,
		userID,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Location{}, fmt.Errorf("location id=%d: %w", id, core_errors.ErrNotFound)
		}
		return domain.Location{}, fmt.Errorf("scan location: %w", err)
	}

	locationDomain := locationDomainFromModel(locationModel)

	return locationDomain, nil
}

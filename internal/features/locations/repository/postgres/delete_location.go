package locations_repository_postgres

import (
	"context"
	"fmt"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func (r *LocationsRepository) DeleteLocation(
	ctx context.Context,
	id int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	DELETE FROM weather.locations
	WHERE id=$1;
	`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("location with id='%d': %w", id, core_errors.ErrNotFound)
	}

	return nil
}

package locations_service

import (
	"context"
	"fmt"
)

func (s *LocationsService) DeleteLocation(
	ctx context.Context,
	id int64,
) error {
	err := s.locationsRepository.DeleteLocation(ctx, id)
	if err != nil {
		return fmt.Errorf("delete location in repository: %w", err)
	}

	return nil
}

package domain

type Location struct {
	ID        int64
	UserID    int64
	Name      string
	Latitude  float64
	Longitude float64
}

func NewLocation(
	id int64,
	userID int64,
	name string,
	latitude float64,
	longitude float64,
) Location {
	return Location{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Latitude:  latitude,
		Longitude: longitude,
	}
}

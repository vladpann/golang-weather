package core_postgres_pool

import "errors"

var (
	ErrNoRows             = errors.New("no rows")
	ErrViolatesForeignKey = errors.New("violates foreign key")
	ErrUniqueViolation    = errors.New("unique violation")
	ErrUnknown            = errors.New("unknown")
)

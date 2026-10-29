package core_errors

import "errors"

var (
	ErrNotFound              = errors.New("not found")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrConflict              = errors.New("conflict")
	ErrLoginAlreadyExists    = errors.New("login already exists")
	ErrPasswordDoNotMatch    = errors.New("password do not match")
	ErrLocationAlreadyExists = errors.New("location already exists")
	ErrLoginRequired         = errors.New("login is required")
	ErrPasswordRequired      = errors.New("password is required")
	ErrPasswordConstrains    = errors.New("password must be between 4 and 72 characters")
	ErrCityNameRequired      = errors.New("city name is required")
)

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
)

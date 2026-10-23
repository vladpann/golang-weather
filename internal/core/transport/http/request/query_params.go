package core_http_request

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

var validate = validator.New()

func GetIntQueryParam(r *http.Request, key string) (*int, error) {
	param := r.URL.Query().Get(key)
	if param == "" {
		return nil, nil
	}

	val, err := strconv.Atoi(param)
	if err != nil {
		return nil, fmt.Errorf(
			"param='%s' by key='%s' not a valid integer: %v: %w",
			param,
			key,
			err,
			core_errors.ErrInvalidArgument,
		)
	}

	return &val, nil
}

func GetFloat64QueryParam(r *http.Request, key string) (float64, error) {
	param := r.URL.Query().Get(key)
	if param == "" {
		return 0, fmt.Errorf(
			"query parameter '%s' is required: %w",
			key,
			core_errors.ErrInvalidArgument,
		)
	}

	val, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"param='%s' by key='%s' not a valid float: %v: %w",
			param,
			key,
			err,
			core_errors.ErrInvalidArgument,
		)
	}

	return val, nil
}

// ParseQueryParams parses query parameters from request and validates them
func ParseQueryParams(r *http.Request, target interface{}) error {
	// This is a simplified implementation that works with struct tags
	// For more complex scenarios, consider using a library like gorilla/schema

	// Use reflection to parse query params based on struct tags
	// For now, we'll return an error that suggests using manual parsing
	return fmt.Errorf("ParseQueryParams not fully implemented, use GetFloat64QueryParam: %w", core_errors.ErrInvalidArgument)
}

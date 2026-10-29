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

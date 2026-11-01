package core_http_request

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

var validate = validator.New()

const (
	DefaultPaginationLimit = 10
	MaxPaginationLimit     = 100
)

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

	if val < 0 {
		return nil, fmt.Errorf(
			"param='%s' by key='%s' must be non-negative: %w",
			param,
			key,
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

	if math.IsInf(val, 0) {
		return 0, fmt.Errorf(
			"param='%s' by key='%s' is infinite: %w",
			param,
			key,
			core_errors.ErrInvalidArgument,
		)
	}

	if math.IsNaN(val) {
		return 0, fmt.Errorf(
			"param='%s' by key='%s' is NaN: %w",
			param,
			key,
			core_errors.ErrInvalidArgument,
		)
	}

	return val, nil
}

func GetPaginationParams(r *http.Request) (limit int, offset int, err error) {
	limitPtr, err := GetIntQueryParam(r, "limit")
	if err != nil {
		return 0, 0, fmt.Errorf("get 'limit' query param: %w", err)
	}

	offsetPtr, err := GetIntQueryParam(r, "offset")
	if err != nil {
		return 0, 0, fmt.Errorf("get 'offset' query param: %w", err)
	}

	if limitPtr == nil {
		limit = DefaultPaginationLimit
	} else {
		limit = *limitPtr
	}

	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}

	if offsetPtr == nil {
		offset = 0
	} else {
		offset = *offsetPtr
	}

	return limit, offset, nil
}

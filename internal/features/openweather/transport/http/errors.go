package openweather_transport_http

import (
	"errors"
	"fmt"
	"net/url"
)

// sanitizeRequestError removes the URL (and, with it, the appid) from net/http errors
func sanitizeRequestError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("openweather request %s failed: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

package types

import (
	"errors"
	"fmt"
)

const (
	// NotFoundCode is code for the error of no object found
	NotFoundCode = "NOT_FOUND"
	// UnAuthorizedCode ...
	UnAuthorizedCode = "UNAUTHORIZED"
	// ForbiddenCode ...
	ForbiddenCode = "FORBIDDEN"
	// RateLimitCode
	RateLimitCode = "TOO_MANY_REQUEST"
	// GeneralCode ...
	GeneralCode = "UNKNOWN"
)

type Error struct {
	Code       string
	StatusCode int
	Body       string
}

func (e *Error) Error() string {
	return fmt.Sprintf("registry request failed: code=%s status=%d body=%s", e.Code, e.StatusCode, e.Body)
}

func IsCode(err error, code string) bool {
	var registryErr *Error
	return errors.As(err, &registryErr) && registryErr.Code == code
}

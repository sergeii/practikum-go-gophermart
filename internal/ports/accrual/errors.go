package accrual

import (
	"errors"
	"fmt"
)

var ErrConfigInvalidAddress = errors.New("invalid accrual system address")
var ErrRespInvalidWaitTime = errors.New("Retry-After header is missing or has invalid value")
var ErrRespInvalidStatus = errors.New("invalid response from accrual system")
var ErrRespInvalidData = errors.New("unexpected data from accrual system")
var ErrOrderNotFound = errors.New("order not found in accrual system")

type TooManyRequestError struct {
	RetryAfter uint
}

func NewErrTooManyRequests(retryAfter uint) error {
	return &TooManyRequestError{retryAfter}
}

func (err TooManyRequestError) Error() string {
	return fmt.Sprintf("retry after: %d", err.RetryAfter)
}

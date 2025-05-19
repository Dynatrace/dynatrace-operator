package dterror

import (
	"fmt"
	"github.com/pkg/errors"
)

type DtError interface {
	error
	GetErrorCode() string
	Unwrap() error
}

type dtError struct {
	error
	errorCode string
}

func (dtErr *dtError) Error() string {
	return fmt.Sprintf("%s - %s", dtErr.errorCode, dtErr.error.Error())
}

func (dtErr *dtError) Unwrap() error {
	return dtErr.error
}

func New(errorCode string, message string) error {
	return &dtError{errorCode: errorCode, error: errors.New(message)}
}

func Errorf(errorCode string, format string, a ...any) error {
	return &dtError{errorCode: errorCode, error: errors.Errorf(format, a...)}
}

func WithErrorCode(err error, errorCode string) error {
	return &dtError{error: err, errorCode: errorCode}
}

func (dtErr *dtError) GetErrorCode() string {
	return dtErr.errorCode
}

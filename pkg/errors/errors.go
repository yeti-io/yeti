package errors

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound           = NewError("NOT_FOUND", "resource not found", http.StatusNotFound)
	ErrValidation         = NewError("VALIDATION_ERROR", "validation failed", http.StatusBadRequest)
	ErrInternal           = NewError("INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
	ErrConflict           = NewError("CONFLICT", "resource conflict", http.StatusConflict)
	ErrUnauthorized       = NewError("UNAUTHORIZED", "unauthorized", http.StatusUnauthorized)
	ErrForbidden          = NewError("FORBIDDEN", "forbidden", http.StatusForbidden)
	ErrTimeout            = NewError("TIMEOUT", "operation timed out", http.StatusRequestTimeout)
	ErrServiceUnavailable = NewError("SERVICE_UNAVAILABLE", "service unavailable", http.StatusServiceUnavailable)
)

type RetryableError interface {
	error
	IsRetryable() bool
}

type FatalError interface {
	error
	IsFatal() bool
}

type Error struct {
	Code      string
	Message   string
	Status    int
	Details   map[string]interface{}
	Cause     error
	retryable *bool
}

func NewError(code, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
		Details: make(map[string]interface{}),
	}
}

func (e *Error) Error() string {
	msg := e.Message

	if len(e.Details) > 0 {
		if detailMsg, ok := e.Details["message"].(string); ok && detailMsg != "" {
			msg = detailMsg
		}
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, msg, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, msg)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) IsRetryable() bool {
	if e.retryable != nil {
		return *e.retryable
	}
	if e.Cause != nil {
		var retryableErr RetryableError
		if errors.As(e.Cause, &retryableErr) {
			return retryableErr.IsRetryable()
		}
		var fatalErr FatalError
		if errors.As(e.Cause, &fatalErr) {
			return !fatalErr.IsFatal()
		}
	}
	return e.Code != ErrValidation.Code && e.Code != ErrNotFound.Code
}

func (e *Error) IsFatal() bool {
	if e.retryable != nil {
		return !*e.retryable
	}

	if e.Cause != nil {
		var fatalErr FatalError
		if errors.As(e.Cause, &fatalErr) {
			return fatalErr.IsFatal()
		}
	}

	return e.Code == ErrValidation.Code || e.Code == ErrNotFound.Code
}

func (e *Error) WithCause(cause error) *Error {
	err := *e
	err.Cause = cause
	return &err
}

func (e *Error) WithDetail(key string, value interface{}) *Error {
	err := *e
	if err.Details == nil {
		err.Details = make(map[string]interface{})
	}
	err.Details[key] = value
	return &err
}

func (e *Error) WithDetails(details map[string]interface{}) *Error {
	err := *e
	err.Details = details
	return &err
}

func (e *Error) AsRetryable() *Error {
	err := *e
	retryable := true
	err.retryable = &retryable
	return &err
}

func (e *Error) AsFatal() *Error {
	err := *e
	retryable := false
	err.retryable = &retryable
	return &err
}

func Wrap(err error, appErr *Error) *Error {
	if err == nil {
		return nil
	}
	return appErr.WithCause(err)
}

func IsNotFound(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == ErrNotFound.Code
	}
	return false
}

func IsValidation(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == ErrValidation.Code
	}
	return false
}

func IsConflict(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == ErrConflict.Code
	}
	return false
}

func ToHTTPStatus(err error) int {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return http.StatusInternalServerError
}

func ToErrorResponse(err error) map[string]interface{} {
	var appErr *Error
	if !errors.As(err, &appErr) {
		// If it's not our error type, wrap it
		appErr = ErrInternal.WithCause(err)
	}

	response := map[string]interface{}{
		"error":      appErr.Message,
		"error_code": appErr.Code,
	}

	if len(appErr.Details) > 0 {
		response["details"] = appErr.Details
	}

	return response
}

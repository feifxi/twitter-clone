package apperr

import (
	"errors"
)

type Kind string

const (
	KindBadRequest   Kind = "BAD_REQUEST"
	KindUnauthorized Kind = "UNAUTHORIZED"
	KindForbidden    Kind = "FORBIDDEN"
	KindNotFound     Kind = "NOT_FOUND"
	KindConflict     Kind = "CONFLICT"
	KindInternal     Kind = "INTERNAL"
)

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newError(kind Kind, message string, err error) *Error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func BadRequest(message string) *Error {
	return newError(KindBadRequest, message, nil)
}

func Unauthorized(message string) *Error {
	return newError(KindUnauthorized, message, nil)
}

func Forbidden(message string) *Error {
	return newError(KindForbidden, message, nil)
}

func NotFound(message string) *Error {
	return newError(KindNotFound, message, nil)
}

func Conflict(message string) *Error {
	return newError(KindConflict, message, nil)
}

func Internal(message string, err error) *Error {
	if err == nil {
		err = errors.New(message)
	}
	return newError(KindInternal, message, err)
}

func KindOf(err error) (Kind, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Kind, true
	}
	return "", false
}

func MessageOf(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return ""
}

// ErrorResponse is the standard JSON error envelope returned by HTTP handlers.
type ErrorResponse struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"requestId,omitempty"`
	Details   []FieldError `json:"details,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

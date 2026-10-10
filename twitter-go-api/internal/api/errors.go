package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email format"
	case "min", "max":
		bound := map[string]string{"min": "at least", "max": "at most"}[fe.Tag()]
		switch fe.Kind() {
		case reflect.String:
			return fmt.Sprintf("must be %s %s characters", bound, fe.Param())
		case reflect.Slice, reflect.Array, reflect.Map:
			return fmt.Sprintf("must have %s %s items", bound, fe.Param())
		default:
			return fmt.Sprintf("must be %s %s", bound, fe.Param())
		}
	case "http_url":
		return "must be a full http(s) URL"
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	case "numeric":
		return "must be a valid number"
	case "required_without":
		return fmt.Sprintf("this field is required if %s is not provided", fe.Param())
	}
	return "invalid value"
}

func writeError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	requestID := middleware.GetRequestID(c)

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		out := make([]apperr.FieldError, len(validationErrs))
		for i, fe := range validationErrs {
			out[i] = apperr.FieldError{
				Field:   fe.Field(),
				Message: msgForTag(fe),
			}
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:      "VALIDATION_ERROR",
			Message:   "invalid request payload",
			RequestID: requestID,
			Details:   out,
		})
		return
	}
	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		response := ErrorResponse{
			Code:      "VALIDATION_ERROR",
			Message:   "invalid request payload",
			RequestID: requestID,
		}
		if field, msg := inferNumericFieldError(c); field != "" {
			response.Details = []apperr.FieldError{{Field: field, Message: msg}}
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:      "BAD_REQUEST",
			Message:   "malformed JSON request body",
			RequestID: requestID,
		})
		return
	}

	status, code, message := httpError(err)
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed", "err", err, "method", c.Request.Method, "route", c.FullPath())
	}
	c.JSON(status, ErrorResponse{Code: code, Message: message, RequestID: requestID})
}

type errorMapping struct {
	status  int
	code    string
	message string // default when the error carries none; always used for 5xx
}

var kindMappings = map[apperr.Kind]errorMapping{
	apperr.KindBadRequest:   {http.StatusBadRequest, "BAD_REQUEST", "invalid request"},
	apperr.KindUnauthorized: {http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"},
	apperr.KindForbidden:    {http.StatusForbidden, "FORBIDDEN", "forbidden"},
	apperr.KindNotFound:     {http.StatusNotFound, "NOT_FOUND", "resource not found"},
	apperr.KindConflict:     {http.StatusConflict, "CONFLICT", "conflict"},
}

// Postgres constraint violations that reach the handler are client errors, not 500s.
var pgMappings = map[string]errorMapping{
	"23505": {http.StatusConflict, "CONFLICT", "resource already exists"},                 // unique_violation
	"23503": {http.StatusBadRequest, "BAD_REQUEST", "referenced resource does not exist"}, // foreign_key_violation
	"23514": {http.StatusBadRequest, "BAD_REQUEST", "invalid request"},                    // check_violation
}

var internalError = errorMapping{http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"}

// httpError maps a service error to a response. Only apperr messages are shown
// to clients; anything unrecognised becomes an opaque 500.
func httpError(err error) (status int, code, message string) {
	if kind, ok := apperr.KindOf(err); ok {
		if m, ok := kindMappings[kind]; ok {
			if msg := apperr.MessageOf(err); msg != "" {
				return m.status, m.code, msg
			}
			return m.status, m.code, m.message
		}
		return internalError.status, internalError.code, internalError.message
	}

	if errors.Is(err, pgx.ErrNoRows) {
		m := kindMappings[apperr.KindNotFound]
		return m.status, m.code, m.message
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if m, ok := pgMappings[pgErr.Code]; ok {
			return m.status, m.code, m.message
		}
	}
	return internalError.status, internalError.code, internalError.message
}

func writeValidationError(c *gin.Context, field, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:      "VALIDATION_ERROR",
		Message:   "invalid request payload",
		RequestID: middleware.GetRequestID(c),
		Details: []apperr.FieldError{
			{Field: field, Message: message},
		},
	})
}

func inferNumericFieldError(c *gin.Context) (string, string) {
	if c == nil || c.Request == nil {
		return "", ""
	}

	candidates := []string{"parentId", "id", "cursor", "size", "limit"}
	for _, field := range candidates {
		value := strings.TrimSpace(numericFieldRawValue(c, field))
		if value == "" {
			continue
		}
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return field, "must be a valid number"
		}
	}

	return "", ""
}

func numericFieldRawValue(c *gin.Context, field string) string {
	if v := strings.TrimSpace(c.Param(field)); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.Query(field)); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.PostForm(field)); v != "" {
		return v
	}
	return ""
}

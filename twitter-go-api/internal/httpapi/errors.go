package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	zlog "github.com/rs/zerolog/log"
)

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email format"
	case "min":
		return fmt.Sprintf("must be at least %s units", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s units", fe.Param())
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

	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "internal server error"

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			status = http.StatusConflict
			code = "CONFLICT"
			message = "resource already exists"
		case "23503": // foreign_key_violation
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
			message = "referenced resource does not exist"
		case "23514": // check_violation
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
			message = "invalid request"
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
		code = "NOT_FOUND"
		message = "resource not found"
	} else if kind, ok := apperr.KindOf(err); ok {
		switch kind {
		case apperr.KindBadRequest:
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
			message = defaultMessage(apperr.MessageOf(err), "invalid request")
		case apperr.KindUnauthorized:
			status = http.StatusUnauthorized
			code = "UNAUTHORIZED"
			message = defaultMessage(apperr.MessageOf(err), "authentication required")
		case apperr.KindForbidden:
			status = http.StatusForbidden
			code = "FORBIDDEN"
			message = defaultMessage(apperr.MessageOf(err), "forbidden")
		case apperr.KindNotFound:
			status = http.StatusNotFound
			code = "NOT_FOUND"
			message = defaultMessage(apperr.MessageOf(err), "resource not found")
		case apperr.KindConflict:
			status = http.StatusConflict
			code = "CONFLICT"
			message = defaultMessage(apperr.MessageOf(err), "conflict")
		case apperr.KindInternal:
			status = http.StatusInternalServerError
			code = "INTERNAL_ERROR"
			message = "internal server error"
		}
	}

	if status >= 500 {
		zlog.Error().Err(err).
			Str("method", c.Request.Method).
			Str("path", c.FullPath()).
			Str("request_id", requestID).
			Msg("Internal server error")
	}

	c.JSON(status, ErrorResponse{Code: code, Message: message, RequestID: requestID})
}

func defaultMessage(in, fallback string) string {
	if in == "" {
		return fallback
	}
	return in
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

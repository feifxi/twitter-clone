package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func newErrorTestContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, path, bytes.NewReader(nil))
	ctx.Request = req
	return ctx, w
}

func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var out ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode error response: %v body=%s", err, rec.Body.String())
	}
	return out
}

func TestWriteErrorMapsCoreErrorTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		err            error
		path           string
		wantStatus     int
		wantCode       string
		wantMessage    string
		wantDetailSize int
	}{
		{
			name:        "app unauthorized",
			err:         apperr.Unauthorized("missing auth"),
			path:        "/api/v1/test",
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "UNAUTHORIZED",
			wantMessage: "missing auth",
		},
		{
			name:        "not found",
			err:         pgx.ErrNoRows,
			path:        "/api/v1/test",
			wantStatus:  http.StatusNotFound,
			wantCode:    "NOT_FOUND",
			wantMessage: "resource not found",
		},
		{
			name:        "pg unique violation",
			err:         &pgconn.PgError{Code: "23505"},
			path:        "/api/v1/test",
			wantStatus:  http.StatusConflict,
			wantCode:    "CONFLICT",
			wantMessage: "resource already exists",
		},
		{
			name:        "pg foreign key violation",
			err:         &pgconn.PgError{Code: "23503"},
			path:        "/api/v1/test",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "BAD_REQUEST",
			wantMessage: "referenced resource does not exist",
		},
		{
			name:        "pg check violation",
			err:         &pgconn.PgError{Code: "23514"},
			path:        "/api/v1/test",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "BAD_REQUEST",
			wantMessage: "invalid request",
		},
		{
			name:        "internal app error hides message",
			err:         apperr.Internal("sensitive details", nil),
			path:        "/api/v1/test",
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "INTERNAL_ERROR",
			wantMessage: "internal server error",
		},
		{
			name:           "numeric parse error includes details",
			err:            &strconv.NumError{Func: "ParseInt", Num: "abc", Err: strconv.ErrSyntax},
			path:           "/api/v1/test?size=abc",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "VALIDATION_ERROR",
			wantMessage:    "invalid request payload",
			wantDetailSize: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, rec := newErrorTestContext(tc.path)

			writeError(ctx, tc.err)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status mismatch: got %d want %d", rec.Code, tc.wantStatus)
			}

			got := decodeErrorResponse(t, rec)
			if got.Code != tc.wantCode {
				t.Fatalf("code mismatch: got %q want %q", got.Code, tc.wantCode)
			}
			if got.Message != tc.wantMessage {
				t.Fatalf("message mismatch: got %q want %q", got.Message, tc.wantMessage)
			}
			if len(got.Details) != tc.wantDetailSize {
				t.Fatalf("details size mismatch: got %d want %d", len(got.Details), tc.wantDetailSize)
			}
		})
	}
}

func TestWriteErrorValidationErrors(t *testing.T) {
	t.Parallel()

	v := validator.New()
	type req struct {
		Name string `validate:"required"`
	}
	err := v.Struct(req{})

	ctx, rec := newErrorTestContext("/api/v1/test")
	writeError(ctx, err)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	got := decodeErrorResponse(t, rec)
	if got.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", got.Code)
	}
	if len(got.Details) == 0 {
		t.Fatal("expected validation details")
	}
}

func TestWriteValidationError(t *testing.T) {
	t.Parallel()

	ctx, rec := newErrorTestContext("/api/v1/test")
	writeValidationError(ctx, "media", "unsupported media type")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	got := decodeErrorResponse(t, rec)
	if got.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", got.Code)
	}
	if len(got.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(got.Details))
	}
	if got.Details[0].Field != "media" || got.Details[0].Message != "unsupported media type" {
		t.Fatalf("unexpected detail: %+v", got.Details[0])
	}
}

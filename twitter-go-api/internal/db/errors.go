package db

import (
	"errors"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// NotFound turns pgx.ErrNoRows into a not-found error naming the resource
// ("tweet not found"); any other error, or nil, passes through unchanged.
func NotFound(err error, resource string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound(resource + " not found")
	}
	return err
}

// IsUniqueViolation reports whether err is a Postgres unique_violation (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

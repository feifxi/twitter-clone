package db

import (
	"errors"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/jackc/pgx/v5"
)

// NotFound turns pgx.ErrNoRows into a not-found error naming the resource
// ("tweet not found"); any other error, or nil, passes through unchanged.
func NotFound(err error, resource string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound(resource + " not found")
	}
	return err
}

package feed_test

import "github.com/jackc/pgx/v5"

func pgxNoRows() error { return pgx.ErrNoRows }

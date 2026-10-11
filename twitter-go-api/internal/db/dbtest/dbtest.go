// Package dbtest provides a hook-based fake of db.Store for unit tests.
package dbtest

//go:generate go run ./internal/genmock ../querier.go mock_store.go

// Ptr returns a pointer to v (handy for nullable sqlc fields).
func Ptr[T any](v T) *T { return &v }

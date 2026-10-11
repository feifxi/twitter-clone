package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	ExecTx(ctx context.Context, fn func(Querier) error) error
	ExecTxAfterCommit(ctx context.Context, fn func(Querier) error, afterCommit func()) error
	Ping(ctx context.Context) error
}

type SQLStore struct {
	*Queries
	conn *pgxpool.Pool
}

func NewStore(conn *pgxpool.Pool) Store {
	return &SQLStore{
		Queries: New(conn),
		conn:    conn,
	}
}

func (s *SQLStore) ExecTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ExecTxAfterCommit executes fn inside a transaction and only runs afterCommit
// if the transaction has successfully committed.
func (s *SQLStore) ExecTxAfterCommit(ctx context.Context, fn func(Querier) error, afterCommit func()) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if afterCommit != nil {
		afterCommit()
	}
	return nil
}

func (s *SQLStore) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

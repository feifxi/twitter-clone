package dbtest

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var migrateOnce sync.Once

// OpenTestDB returns a store backed by the Postgres at TEST_DATABASE_URL, with
// migrations applied. Tests calling it are skipped when the variable is unset.
// Tests share the database, so they must create their own rows (see NewUser).
func OpenTestDB(t testing.TB) db.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database integration test")
	}

	var migrateErr error
	migrateOnce.Do(func() { migrateErr = db.Migrate(url) })
	if migrateErr != nil {
		t.Fatalf("migrate test database: %v", migrateErr)
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return db.NewStore(pool)
}

// NewUser inserts a user with a unique username and email.
func NewUser(t testing.TB, store db.Querier) db.User {
	t.Helper()
	suffix := strings.ToLower(rand.Text()[:10])
	u, err := store.CreateUser(context.Background(), db.CreateUserParams{
		Username: "u_" + suffix,
		Email:    suffix + "@example.test",
		Role:     "USER",
		Provider: "TEST",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// NewTweet inserts a top-level text tweet.
func NewTweet(t testing.TB, store db.Querier, userID int64, content string) db.Tweet {
	t.Helper()
	tw, err := store.CreateTweet(context.Background(), db.CreateTweetParams{UserID: userID, Content: &content})
	if err != nil {
		t.Fatalf("create tweet: %v", err)
	}
	return tw
}

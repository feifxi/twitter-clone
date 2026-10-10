package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestConsumeRefreshToken_WorksExactlyOnceUnderRace(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	u := dbtest.NewUser(t, store)

	_, err := store.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{UserID: u.ID, TokenHash: "race-" + u.Username, ExpiryDate: time.Now().Add(time.Hour)})
	require.NoError(t, err)

	const attempts = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for range attempts {
		wg.Go(func() {
			_, err := store.ConsumeRefreshToken(ctx, "race-"+u.Username)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				succeeded++
			} else if !errors.Is(err, pgx.ErrNoRows) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	wg.Wait()
	require.Equal(t, 1, succeeded, "a refresh token must be usable exactly once")
}

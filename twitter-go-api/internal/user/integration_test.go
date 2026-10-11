package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/stretchr/testify/require"
)

func TestUsernames_AreCaseInsensitive(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	first, second := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	svc := user.New(store, nil, nil)
	handle := "Case_" + first.Username[2:9]

	onboarded, err := svc.CompleteOnboarding(ctx, first.ID, handle, "First")
	require.NoError(t, err)
	require.True(t, onboarded.Onboarded)

	_, err = svc.CompleteOnboarding(ctx, second.ID, strings.ToLower(handle), "Second")
	kind, _ := apperr.KindOf(err)
	require.Equal(t, apperr.KindConflict, kind, "usernames differing only by case collide")

	found, err := svc.GetByUsername(ctx, strings.ToUpper(handle), nil)
	require.NoError(t, err)
	require.Equal(t, first.ID, found.ID)
}

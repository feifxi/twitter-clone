package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
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

func TestSuggestedUsers_OnlyOnboardedAccounts(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()

	viewer := dbtest.NewUser(t, store)
	pending := dbtest.NewUser(t, store) // still on its generated handle
	onboarded := dbtest.NewUser(t, store)
	name := "Onboarded"
	_, err := store.SetUsername(ctx, db.SetUsernameParams{ID: onboarded.ID, Username: onboarded.Username, DisplayName: &name})
	require.NoError(t, err)

	rows, err := store.ListSuggestedUsers(ctx, db.ListSuggestedUsersParams{ViewerID: &viewer.ID, PageLimit: 100000})
	require.NoError(t, err)
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.User.ID] = true
	}
	require.True(t, ids[onboarded.ID], "onboarded account is suggested")
	require.False(t, ids[pending.ID], "account without a chosen username is not suggested")
	require.False(t, ids[viewer.ID], "never suggest yourself")
}

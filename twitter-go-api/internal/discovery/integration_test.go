package discovery_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/stretchr/testify/require"
)

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

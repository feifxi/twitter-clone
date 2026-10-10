package usecase

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
)

type UpdateProfileInput struct {
	Bio         *string
	DisplayName *string
	AvatarKey   *string // nil = unchanged, "" = remove, otherwise a key from a presigned upload
	BannerKey   *string // same semantics as AvatarKey
}

func (u *UserUsecase) GetUser(ctx context.Context, targetUserID int64, viewerID *int64) (UserItem, error) {
	user, err := u.store.GetUser(ctx, db.GetUserParams{ID: targetUserID, ViewerID: viewerID})
	if err != nil {
		return UserItem{}, err
	}

	return newUserItemFromDB(user.User, user.IsFollowing), nil
}

func (u *UserUsecase) UpdateProfile(ctx context.Context, userID int64, input UpdateProfileInput) (UserItem, error) {
	existing, err := u.store.GetUser(ctx, db.GetUserParams{ID: userID})
	if err != nil {
		return UserItem{}, err
	}
	current := existing.User

	avatar, err := u.resolveProfileImage(current.AvatarUrl, input.AvatarKey, FolderAvatars, userID)
	if err != nil {
		return UserItem{}, err
	}
	banner, err := u.resolveProfileImage(current.BannerUrl, input.BannerKey, FolderBanners, userID)
	if err != nil {
		return UserItem{}, err
	}

	bio := current.Bio
	if input.Bio != nil {
		bio = input.Bio
	}
	displayName := current.DisplayName
	if input.DisplayName != nil {
		displayName = input.DisplayName
	}

	updated, err := u.store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:          userID,
		Bio:         bio,
		DisplayName: displayName,
		AvatarUrl:   avatar,
		BannerUrl:   banner,
	})
	if err != nil {
		u.deleteNewUploads(ctx, input.AvatarKey, input.BannerKey)
		return UserItem{}, err
	}

	if input.AvatarKey != nil {
		u.deleteHostedURL(ctx, current.AvatarUrl)
	}
	if input.BannerKey != nil {
		u.deleteHostedURL(ctx, current.BannerUrl)
	}

	return newUserItemFromDB(updated, false), nil
}

func (u *UserUsecase) resolveProfileImage(current, key *string, folder string, userID int64) (*string, error) {
	if key == nil {
		return current, nil
	}
	if *key == "" {
		return nil, nil
	}
	if err := validateOwnedKey(*key, folder, userID); err != nil {
		return nil, err
	}
	url := u.storage.PublicURL(*key)
	return &url, nil
}

func (u *UserUsecase) deleteNewUploads(ctx context.Context, keys ...*string) {
	for _, key := range keys {
		if key != nil && *key != "" {
			_ = u.storage.DeleteObject(ctx, *key)
		}
	}
}

// deleteHostedURL only removes objects in our bucket; external URLs (e.g. Google avatars) are left alone.
func (u *UserUsecase) deleteHostedURL(ctx context.Context, url *string) {
	if url == nil {
		return
	}
	if key, ok := u.storage.KeyFromPublicURL(*url); ok {
		_ = u.storage.DeleteObject(ctx, key)
	}
}

func (u *UserUsecase) FollowUser(ctx context.Context, followerID, targetUserID int64) (bool, error) {
	if followerID == targetUserID {
		return false, apperr.BadRequest("cannot follow yourself")
	}

	targetUser, err := u.store.GetUser(ctx, db.GetUserParams{ID: targetUserID, ViewerID: nil})
	if err != nil {
		return false, err
	}

	var inserted bool
	var pendingNotification db.Notification
	err = u.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		var err error
		inserted, err = q.FollowUser(ctx, db.FollowUserParams{FollowerID: followerID, FollowingID: targetUserID})
		if err != nil {
			return err
		}

		if inserted {
			pendingNotification, err = createNotification(ctx, q, targetUser.User.ID, followerID, nil, NotifTypeFollow)
		}
		return err
	}, func() {
		if pendingNotification.ID != 0 {
			dispatchNotification(u.publishNotification, pendingNotification)
		}
	})
	if err != nil {
		return false, err
	}

	return inserted, nil
}

func (u *UserUsecase) UnfollowUser(ctx context.Context, followerID, targetUserID int64) error {
	if _, err := u.store.GetUser(ctx, db.GetUserParams{ID: targetUserID, ViewerID: nil}); err != nil {
		return err
	}

	_, err := u.store.UnfollowUser(ctx, db.UnfollowUserParams{FollowerID: followerID, FollowingID: targetUserID})
	return err
}

func (u *UserUsecase) ListFollowers(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]UserItem, error) {
	if _, err := u.store.GetUser(ctx, db.GetUserParams{ID: targetUserID, ViewerID: viewerID}); err != nil {
		return nil, err
	}

	users, err := u.store.ListFollowersUsers(ctx, db.ListFollowersUsersParams{
		FollowingID: targetUserID,
		Limit:       size,
		Offset:      page * size,
		ViewerID:    viewerID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]UserItem, 0, len(users))
	for _, r := range users {
		items = append(items, newUserItemFromDB(r.User, r.IsFollowing))
	}
	return items, nil
}

func (u *UserUsecase) ListFollowing(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]UserItem, error) {
	if _, err := u.store.GetUser(ctx, db.GetUserParams{ID: targetUserID, ViewerID: viewerID}); err != nil {
		return nil, err
	}

	users, err := u.store.ListFollowingUsers(ctx, db.ListFollowingUsersParams{
		FollowerID: targetUserID,
		Limit:      size,
		Offset:     page * size,
		ViewerID:   viewerID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]UserItem, 0, len(users))
	for _, r := range users {
		items = append(items, newUserItemFromDB(r.User, r.IsFollowing))
	}
	return items, nil
}

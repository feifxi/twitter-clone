// Package user manages profiles and the follow graph.
package user

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/notification"
)

type objectStore interface {
	PublicURL(objectKey string) string
	KeyFromPublicURL(url string) (string, bool)
	DeleteObject(ctx context.Context, objectKey string) error
}

type notifier interface {
	Dispatch(n db.Notification)
}

type Service struct {
	store    db.Store
	storage  objectStore
	notifier notifier
}

func New(store db.Store, storage objectStore, notifier notifier) *Service {
	return &Service{store: store, storage: storage, notifier: notifier}
}

type UpdateProfileInput struct {
	Bio         *string
	DisplayName *string
	AvatarKey   *string // nil = unchanged, "" = remove, otherwise a key from a presigned upload
	BannerKey   *string // same semantics as AvatarKey
}

func (s *Service) Get(ctx context.Context, userID int64, viewerID *int64) (domain.User, error) {
	row, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: viewerID})
	if err != nil {
		return domain.User{}, err
	}
	return domain.NewUser(row.User, row.IsFollowing), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, in UpdateProfileInput) (domain.User, error) {
	existing, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID})
	if err != nil {
		return domain.User{}, err
	}
	current := existing.User

	avatar, err := s.resolveProfileImage(current.AvatarURL, in.AvatarKey, media.FolderAvatars, userID)
	if err != nil {
		return domain.User{}, err
	}
	banner, err := s.resolveProfileImage(current.BannerURL, in.BannerKey, media.FolderBanners, userID)
	if err != nil {
		return domain.User{}, err
	}

	bio := current.Bio
	if in.Bio != nil {
		bio = in.Bio
	}
	displayName := current.DisplayName
	if in.DisplayName != nil {
		displayName = in.DisplayName
	}

	updated, err := s.store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:          userID,
		Bio:         bio,
		DisplayName: displayName,
		AvatarURL:   avatar,
		BannerURL:   banner,
	})
	if err != nil {
		s.deleteNewUploads(ctx, in.AvatarKey, in.BannerKey)
		return domain.User{}, err
	}

	if in.AvatarKey != nil {
		s.deleteHostedURL(ctx, current.AvatarURL)
	}
	if in.BannerKey != nil {
		s.deleteHostedURL(ctx, current.BannerURL)
	}

	return domain.NewUser(updated, false), nil
}

func (s *Service) resolveProfileImage(current, key *string, folder string, userID int64) (*string, error) {
	if key == nil {
		return current, nil
	}
	if *key == "" {
		return nil, nil
	}
	if err := media.ValidateOwnedKey(*key, folder, userID); err != nil {
		return nil, err
	}
	url := s.storage.PublicURL(*key)
	return &url, nil
}

func (s *Service) deleteNewUploads(ctx context.Context, keys ...*string) {
	for _, key := range keys {
		if key != nil && *key != "" {
			_ = s.storage.DeleteObject(ctx, *key)
		}
	}
}

// deleteHostedURL only removes objects in our bucket; external URLs (e.g. Google avatars) are left alone.
func (s *Service) deleteHostedURL(ctx context.Context, url *string) {
	if url == nil {
		return
	}
	if key, ok := s.storage.KeyFromPublicURL(*url); ok {
		_ = s.storage.DeleteObject(ctx, key)
	}
}

// Follow reports whether a new follow was created (false if it already existed).
func (s *Service) Follow(ctx context.Context, followerID, targetID int64) (bool, error) {
	if followerID == targetID {
		return false, apperr.BadRequest("cannot follow yourself")
	}
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: targetID}); err != nil {
		return false, err
	}

	var inserted bool
	var pending db.Notification
	err := s.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		var err error
		inserted, err = q.FollowUser(ctx, db.FollowUserParams{FollowerID: followerID, FollowingID: targetID})
		if err != nil || !inserted {
			return err
		}
		pending, err = notification.Create(ctx, q, targetID, followerID, nil, domain.NotificationFollow)
		return err
	}, func() { s.notifier.Dispatch(pending) })
	return inserted, err
}

func (s *Service) Unfollow(ctx context.Context, followerID, targetID int64) error {
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: targetID}); err != nil {
		return err
	}
	_, err := s.store.UnfollowUser(ctx, db.UnfollowUserParams{FollowerID: followerID, FollowingID: targetID})
	return err
}

func (s *Service) Followers(ctx context.Context, userID int64, page, size int32, viewerID *int64) ([]domain.User, error) {
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: viewerID}); err != nil {
		return nil, err
	}

	rows, err := s.store.ListFollowersUsers(ctx, db.ListFollowersUsersParams{
		FollowingID: userID,
		Limit:       size,
		Offset:      page * size,
		ViewerID:    viewerID,
	})
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, domain.NewUser(r.User, r.IsFollowing))
	}
	return users, nil
}

func (s *Service) Following(ctx context.Context, userID int64, page, size int32, viewerID *int64) ([]domain.User, error) {
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: viewerID}); err != nil {
		return nil, err
	}

	rows, err := s.store.ListFollowingUsers(ctx, db.ListFollowingUsersParams{
		FollowerID: userID,
		Limit:      size,
		Offset:     page * size,
		ViewerID:   viewerID,
	})
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, domain.NewUser(r.User, r.IsFollowing))
	}
	return users, nil
}

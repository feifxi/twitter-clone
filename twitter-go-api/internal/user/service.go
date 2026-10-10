// Package user manages profiles and the follow graph.
package user

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/pagination"
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

// UpdateProfileInput fields are optional: nil leaves the value unchanged and ""
// clears it. Image keys must come from a presigned upload.
type UpdateProfileInput struct {
	Bio         *string
	DisplayName *string
	Location    *string
	Website     *string
	AvatarKey   *string
	BannerKey   *string
}

func (s *Service) Get(ctx context.Context, userID int64, viewerID *int64) (domain.User, error) {
	row, err := s.getUser(ctx, userID, viewerID)
	if err != nil {
		return domain.User{}, err
	}
	return domain.NewUser(row.User, row.IsFollowing), nil
}

// GetByUsername looks a user up by handle, ignoring case.
func (s *Service) GetByUsername(ctx context.Context, username string, viewerID *int64) (domain.User, error) {
	row, err := s.store.GetUserByUsername(ctx, db.GetUserByUsernameParams{Username: username, ViewerID: viewerID})
	if err != nil {
		return domain.User{}, db.NotFound(err, "user")
	}
	return domain.NewUser(row.User, row.IsFollowing), nil
}

// CompleteOnboarding sets the user's chosen username and display name.
func (s *Service) CompleteOnboarding(ctx context.Context, userID int64, username, displayName string) (domain.User, error) {
	if !ValidUsername(username) {
		return domain.User{}, apperr.BadRequest("username must be 3-20 letters, numbers or underscores")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return domain.User{}, apperr.BadRequest("display name is required")
	}

	u, err := s.store.SetUsername(ctx, db.SetUsernameParams{ID: userID, Username: username, DisplayName: &displayName})
	if db.IsUniqueViolation(err) {
		return domain.User{}, apperr.Conflict("username is already taken")
	}
	if err != nil {
		return domain.User{}, db.NotFound(err, "user")
	}
	return domain.NewUser(u, false), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, in UpdateProfileInput) (domain.User, error) {
	existing, err := s.getUser(ctx, userID, nil)
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

	updated, err := s.store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:          userID,
		Bio:         merge(current.Bio, in.Bio),
		DisplayName: merge(current.DisplayName, in.DisplayName),
		Location:    merge(current.Location, in.Location),
		Website:     merge(current.Website, in.Website),
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
	if _, err := s.getUser(ctx, targetID, nil); err != nil {
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
	if _, err := s.getUser(ctx, targetID, nil); err != nil {
		return err
	}
	_, err := s.store.UnfollowUser(ctx, db.UnfollowUserParams{FollowerID: followerID, FollowingID: targetID})
	return err
}

func (s *Service) Followers(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error) {
	if _, err := s.getUser(ctx, userID, viewerID); err != nil {
		return pagination.Page[domain.User]{}, err
	}
	rows, err := s.store.ListFollowers(ctx, db.ListFollowersParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return pagination.Page[domain.User]{}, err
	}
	return followPage(rows, req.Limit, func(r db.ListFollowersRow) (db.User, bool, time.Time) { return r.User, r.IsFollowing, r.FollowedAt }), nil
}

func (s *Service) Following(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error) {
	if _, err := s.getUser(ctx, userID, viewerID); err != nil {
		return pagination.Page[domain.User]{}, err
	}
	rows, err := s.store.ListFollowing(ctx, db.ListFollowingParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return pagination.Page[domain.User]{}, err
	}
	return followPage(rows, req.Limit, func(r db.ListFollowingRow) (db.User, bool, time.Time) { return r.User, r.IsFollowing, r.FollowedAt }), nil
}

// followPage pages follower/following rows; the cursor is (follow time, user id).
func followPage[T any](rows []T, limit int32, fields func(T) (db.User, bool, time.Time)) pagination.Page[domain.User] {
	rows, more := pagination.Trim(rows, limit)
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		u, isFollowing, _ := fields(r)
		users = append(users, domain.NewUser(u, isFollowing))
	}
	next := pagination.NextFrom(rows, more, func(r T) pagination.Cursor {
		u, _, followedAt := fields(r)
		return pagination.Cursor{Time: followedAt, ID: u.ID}
	})
	return pagination.Page[domain.User]{Items: users, Next: next}
}

func merge(current, update *string) *string {
	if update == nil {
		return current
	}
	if *update == "" {
		return nil
	}
	return update
}

func (s *Service) getUser(ctx context.Context, userID int64, viewerID *int64) (db.GetUserRow, error) {
	row, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: viewerID})
	return row, db.NotFound(err, "user")
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

// ValidUsername mirrors the users.username CHECK constraint.
func ValidUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

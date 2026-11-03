// Package media issues upload URLs and owns the object-key rules every feature
// that stores user media must follow.
package media

import (
	"context"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
)

type Limits struct {
	MaxTweetMediaBytes int64
	MaxAvatarBytes     int64
	MaxBannerBytes     int64
}

type presigner interface {
	PresignUpload(ctx context.Context, objectKey, contentType string, contentLength int64) (string, error)
}

type Service struct {
	storage presigner
	limits  Limits
}

func New(storage presigner, limits Limits) *Service {
	return &Service{storage: storage, limits: limits}
}

type Upload struct {
	URL       string
	ObjectKey string
}

func (s *Service) PresignUpload(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (Upload, error) {
	ctype := strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := extensions[ctype]
	if !ok {
		return Upload{}, apperr.BadRequest("unsupported content type")
	}

	maxSize, ok := s.maxSize(folder)
	if !ok {
		return Upload{}, apperr.BadRequest("unsupported folder")
	}
	if contentLength <= 0 || contentLength > maxSize {
		return Upload{}, apperr.BadRequest("file too large")
	}
	if folder != FolderTweets && !strings.HasPrefix(ctype, "image/") {
		return Upload{}, apperr.BadRequest("profile images must be images")
	}

	key := newObjectKey(folder, userID, ext)
	url, err := s.storage.PresignUpload(ctx, key, ctype, contentLength)
	if err != nil {
		return Upload{}, apperr.Internal("failed to generate upload URL", err)
	}
	return Upload{URL: url, ObjectKey: key}, nil
}

func (s *Service) maxSize(folder string) (int64, bool) {
	switch folder {
	case FolderAvatars:
		return s.limits.MaxAvatarBytes, true
	case FolderBanners:
		return s.limits.MaxBannerBytes, true
	case FolderTweets:
		return s.limits.MaxTweetMediaBytes, true
	default:
		return 0, false
	}
}

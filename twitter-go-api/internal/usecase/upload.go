package usecase

import (
	"context"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
)

type PresignedUpload struct {
	URL       string
	ObjectKey string
}

func (u *UploadUsecase) GeneratePresignedURL(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (PresignedUpload, error) {
	ctype := strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := uploadExtensions[ctype]
	if !ok {
		return PresignedUpload{}, apperr.BadRequest("unsupported content type")
	}

	maxSize, ok := u.maxUploadSize(folder)
	if !ok {
		return PresignedUpload{}, apperr.BadRequest("unsupported folder")
	}
	if contentLength <= 0 || contentLength > maxSize {
		return PresignedUpload{}, apperr.BadRequest("file too large")
	}
	if folder != FolderTweets && !strings.HasPrefix(ctype, "image/") {
		return PresignedUpload{}, apperr.BadRequest("profile images must be images")
	}

	objectKey := newObjectKey(folder, userID, ext)
	url, err := u.storage.PresignUpload(ctx, objectKey, ctype, contentLength)
	if err != nil {
		return PresignedUpload{}, apperr.Internal("failed to generate upload URL", err)
	}
	return PresignedUpload{URL: url, ObjectKey: objectKey}, nil
}

func (u *UploadUsecase) maxUploadSize(folder string) (int64, bool) {
	switch folder {
	case FolderAvatars:
		return u.config.MaxAvatarBytes, true
	case FolderBanners:
		return u.config.MaxBannerBytes, true
	case FolderTweets:
		return u.config.MaxMediaBytes, true
	default:
		return 0, false
	}
}

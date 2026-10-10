package media

import (
	"fmt"
	"path"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/google/uuid"
)

const (
	FolderTweets  = "tweets"
	FolderAvatars = "avatars"
	FolderBanners = "banners"
)

// extensions is the allow-list of uploadable MIME types. The extension is
// chosen by the server, never taken from the client's filename.
var extensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"video/mp4":  ".mp4",
	"video/webm": ".webm",
}

var videoExtensions = map[string]bool{".mp4": true, ".webm": true}

func newObjectKey(folder string, userID int64, ext string) string {
	return fmt.Sprintf("%s/%d/%s%s", folder, userID, uuid.NewString(), ext)
}

// ValidateOwnedKey rejects any key outside {folder}/{userID}/. Without it a user could
// attach, and later trigger deletion of, another user's object or an arbitrary URL.
func ValidateOwnedKey(key, folder string, userID int64) error {
	name, ok := strings.CutPrefix(key, fmt.Sprintf("%s/%d/", folder, userID))
	if !ok || name == "" || strings.ContainsAny(name, `/\`) || !isAllowedExtension(path.Ext(name)) {
		return apperr.BadRequest("invalid media key")
	}
	return nil
}

func isAllowedExtension(ext string) bool {
	for _, allowed := range extensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

// TypeForKey derives the tweet media type from a validated key.
func TypeForKey(key string) string {
	if videoExtensions[path.Ext(key)] {
		return domain.MediaVideo
	}
	return domain.MediaImage
}

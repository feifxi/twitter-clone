package media_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/storage/storagetest"
)

var limits = media.Limits{MaxBannerBytes: 10 << 20, MaxTweetMediaBytes: 100 << 20}

func TestService_PresignUpload(t *testing.T) {
	t.Run("success_key_is_namespaced_by_user", func(t *testing.T) {
		storage := &storagetest.Fake{}

		upload, err := media.New(storage, limits).PresignUpload(context.Background(), 42, "image/png", media.FolderBanners, 5<<20)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.HasPrefix(upload.ObjectKey, "banners/42/") || !strings.HasSuffix(upload.ObjectKey, ".png") {
			t.Fatalf("unexpected object key %q", upload.ObjectKey)
		}
		if storage.PresignedSize != 5<<20 {
			t.Fatalf("expected content length to be signed, got %d", storage.PresignedSize)
		}
	})

	badRequests := map[string]struct {
		contentType string
		folder      string
		size        int64
	}{
		"invalid_mime":           {"application/x-msdos-program", "banners", 1},
		"invalid_folder":         {"image/png", "malicious", 1},
		"file_too_large":         {"image/png", "banners", 11 << 20},
		"missing_size":           {"image/png", "banners", 0},
		"video_as_profile_image": {"video/mp4", "banners", 1},
	}
	for name, tc := range badRequests {
		t.Run(name, func(t *testing.T) {
			_, err := media.New(&storagetest.Fake{}, limits).PresignUpload(context.Background(), 42, tc.contentType, tc.folder, tc.size)
			if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
				t.Fatalf("expected bad request, got %v", err)
			}
		})
	}
}

func TestValidateOwnedKey(t *testing.T) {
	for name, key := range map[string]string{
		"no_folder":         "invalid-no-slash",
		"another_users_key": "tweets/2/0b1c.png",
		"wrong_folder":      "avatars/1/0b1c.png",
		"external_url":      "https://evil.example/tweets/1/x.png",
		"path_traversal":    "tweets/1/../2/x.png",
		"disallowed_ext":    "tweets/1/x.exe",
	} {
		t.Run("rejects_"+name, func(t *testing.T) {
			if err := media.ValidateOwnedKey(key, media.FolderTweets, 1); err == nil {
				t.Fatalf("expected %q to be rejected", key)
			}
		})
	}

	if err := media.ValidateOwnedKey("tweets/1/clip.mp4", media.FolderTweets, 1); err != nil {
		t.Fatalf("expected own key to pass, got %v", err)
	}
	if got := media.TypeForKey("tweets/1/clip.mp4"); got != domain.MediaVideo {
		t.Fatalf("expected VIDEO, got %s", got)
	}
}

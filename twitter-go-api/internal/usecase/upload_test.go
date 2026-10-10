package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/usecase"
)

func TestUploadUsecase_GeneratePresignedURL(t *testing.T) {
	cfg := config.Config{MaxBannerBytes: 10 << 20, MaxMediaBytes: 100 << 20}

	t.Run("success_key_is_namespaced_by_user", func(t *testing.T) {
		storage := &fakeStorage{}
		uc := usecase.NewUploadUsecase(cfg, storage)

		upload, err := uc.GeneratePresignedURL(context.Background(), 42, "image/png", "banners", 5<<20)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.HasPrefix(upload.ObjectKey, "banners/42/") || !strings.HasSuffix(upload.ObjectKey, ".png") {
			t.Fatalf("unexpected object key %q", upload.ObjectKey)
		}
		if storage.presignedSize != 5<<20 {
			t.Fatalf("expected content length to be signed, got %d", storage.presignedSize)
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
			uc := usecase.NewUploadUsecase(cfg, &fakeStorage{})
			_, err := uc.GeneratePresignedURL(context.Background(), 42, tc.contentType, tc.folder, tc.size)
			if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
				t.Fatalf("expected bad request, got %v", err)
			}
		})
	}
}

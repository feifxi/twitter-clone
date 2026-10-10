package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/media"
)

type fakeMedia struct {
	generatePresignedURLFn func(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (media.Upload, error)
}

func (m *fakeMedia) PresignUpload(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (media.Upload, error) {
	return m.generatePresignedURLFn(ctx, userID, contentType, folder, contentLength)
}

func TestPresignUpload_Success(t *testing.T) {
	t.Parallel()

	mock := &fakeMedia{
		generatePresignedURLFn: func(_ context.Context, userID int64, _, _ string, _ int64) (media.Upload, error) {
			if userID != 1 {
				t.Fatalf("expected caller's user id, got %d", userID)
			}
			return media.Upload{URL: "http://presigned.url", ObjectKey: "banners/1/file.png"}, nil
		},
	}

	reqBody := `{"contentType":"image/png","folder":"banners","contentLength":1024}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/uploads/presign", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	s := &Server{svc: Services{Media: mock}}
	s.presignUpload(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestPresignUpload_UsecaseError(t *testing.T) {
	t.Parallel()

	mock := &fakeMedia{
		generatePresignedURLFn: func(context.Context, int64, string, string, int64) (media.Upload, error) {
			return media.Upload{}, apperr.BadRequest("file too large")
		},
	}

	reqBody := `{"contentType":"image/png","folder":"banners","contentLength":999999999}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/uploads/presign", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	s := &Server{svc: Services{Media: mock}}
	s.presignUpload(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad request from service, got %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestPresignUpload_RequiresContentLengthAndKnownFolder(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		`{"contentType":"image/png","folder":"banners"}`,
		`{"contentType":"image/png","folder":"malicious","contentLength":10}`,
	} {
		ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/uploads/presign", bytes.NewBufferString(body), "application/json")
		setAuthorizedUser(ctx, 1)

		s := &Server{}
		s.presignUpload(ctx)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d, body=%s", body, rec.Code, rec.Body.String())
		}
	}
}

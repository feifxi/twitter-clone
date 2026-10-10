package usecase_test

import (
	"context"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/service"
)

const fakeStorageBaseURL = "https://cdn.test"

// fakeStorage mimics S3StorageService URL mapping and records deletions.
type fakeStorage struct {
	presignedKey  string
	presignedSize int64
	deleted       []string
}

var _ service.StorageService = (*fakeStorage)(nil)

func (f *fakeStorage) PresignUpload(_ context.Context, objectKey, _ string, contentLength int64) (string, error) {
	f.presignedKey = objectKey
	f.presignedSize = contentLength
	return fakeStorageBaseURL + "/presigned/" + objectKey, nil
}

func (f *fakeStorage) DeleteObject(_ context.Context, objectKey string) error {
	f.deleted = append(f.deleted, objectKey)
	return nil
}

func (f *fakeStorage) PublicURL(objectKey string) string {
	return fakeStorageBaseURL + "/" + objectKey
}

func (f *fakeStorage) KeyFromPublicURL(url string) (string, bool) {
	key, ok := strings.CutPrefix(url, fakeStorageBaseURL+"/")
	return key, ok && key != ""
}

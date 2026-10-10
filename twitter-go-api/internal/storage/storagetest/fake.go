// Package storagetest provides an in-memory stand-in for storage.S3.
package storagetest

import (
	"context"
	"strings"
)

const BaseURL = "https://cdn.test"

// Fake mimics storage.S3's URL mapping and records presigns and deletions.
type Fake struct {
	PresignedKey  string
	PresignedSize int64
	Deleted       []string
}

func (f *Fake) PresignUpload(_ context.Context, objectKey, _ string, contentLength int64) (string, error) {
	f.PresignedKey = objectKey
	f.PresignedSize = contentLength
	return BaseURL + "/presigned/" + objectKey, nil
}

func (f *Fake) DeleteObject(_ context.Context, objectKey string) error {
	f.Deleted = append(f.Deleted, objectKey)
	return nil
}

func (f *Fake) PublicURL(objectKey string) string {
	return BaseURL + "/" + objectKey
}

func (f *Fake) KeyFromPublicURL(url string) (string, bool) {
	key, ok := strings.CutPrefix(url, BaseURL+"/")
	return key, ok && key != ""
}

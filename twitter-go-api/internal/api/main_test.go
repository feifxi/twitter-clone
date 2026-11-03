package api

import (
	"bytes"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newHandlerTestContext(method, path string, body *bytes.Buffer, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	reqBody := bytes.NewReader(nil)
	if body != nil {
		reqBody = bytes.NewReader(body.Bytes())
	}
	req := httptest.NewRequest(method, path, reqBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	return ctx, w
}

func setAuthorizedUser(ctx *gin.Context, userID int64) {
	ctx.Set(middleware.AuthorizationPayloadKey, &token.Payload{
		ID:        uuid.New(),
		UserID:    userID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	})
}

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type createUploadRequest struct {
	ContentType   string `json:"contentType" binding:"required"`
	Folder        string `json:"folder" binding:"required,oneof=tweets avatars banners"`
	ContentLength int64  `json:"contentLength" binding:"required,min=1"`
}

type UploadResponse struct {
	PresignedURL string `json:"presignedUrl"`
	ObjectKey    string `json:"objectKey"`
}

// createUpload godoc
// @Summary		Create Upload
// @Description	Create a temporary presigned URL to upload a file directly to S3. The upload must use exactly the declared Content-Type and Content-Length.
// @Tags			Uploads
// @Accept			json
// @Produce		json
// @Param			request	body			createUploadRequest	true	"Upload request details"
// @Success		201		{object}	UploadResponse
// @Security		BearerAuth
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Router			/uploads [post]
func (s *Server) createUpload(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req createUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	upload, err := s.svc.Media.PresignUpload(c.Request.Context(), userID, req.ContentType, req.Folder, req.ContentLength)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, UploadResponse{PresignedURL: upload.URL, ObjectKey: upload.ObjectKey})
}

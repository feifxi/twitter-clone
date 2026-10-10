package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type presignRequest struct {
	ContentType   string `json:"contentType" binding:"required"`
	Folder        string `json:"folder" binding:"required,oneof=tweets avatars banners"`
	ContentLength int64  `json:"contentLength" binding:"required,min=1"`
}

type PresignResponse struct {
	PresignedURL string `json:"presignedUrl"`
	ObjectKey    string `json:"objectKey"`
}

// presignUpload godoc
// @Summary		Presign Upload URL
// @Description	Generate a temporary presigned URL to upload a file directly to S3. The upload must use exactly the declared Content-Type and Content-Length.
// @Tags			Uploads
// @Accept			json
// @Produce		json
// @Param			request	body			presignRequest	true	"Upload request details"
// @Success		200		{object}	PresignResponse
// @Security		BearerAuth
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Router			/uploads/presign [post]
func (server *Server) presignUpload(ctx *gin.Context) {
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}

	var req presignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, err)
		return
	}

	upload, err := server.uploadUC.GeneratePresignedURL(ctx, userID, req.ContentType, req.Folder, req.ContentLength)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, PresignResponse{PresignedURL: upload.URL, ObjectKey: upload.ObjectKey})
}

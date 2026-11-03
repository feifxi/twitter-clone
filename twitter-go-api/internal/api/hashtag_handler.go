package api

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/gin-gonic/gin"
)

// getTrendingHashtags godoc
// @Summary		Trending Hashtags
// @Description	Get a list of currently trending hashtags based on recent usage.
// @Tags			Hashtags
// @Produce		json
// @Param			limit	query		int		false	"Max number of results"
// @Success		200		{array}		HashtagResponse
// @Router			/hashtags/trending [get]
func (s *Server) getTrendingHashtags(c *gin.Context) {
	limit := parseLimit(c, 10, pagination.MaxLimit)
	hashtags, err := s.svc.Tweets.TrendingHashtags(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newHashtagResponseList(hashtags)
	c.JSON(http.StatusOK, response)
}

package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type sendMessageRequest struct {
	Content string `json:"content" binding:"required,max=2000"`
}

// listConversations godoc
// @Summary		List Conversations
// @Description	Get a paginated list of the current user's conversations.
// @Tags			Messages
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[ConversationResponse]
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/messages/conversations [get]
func (s *Server) listConversations(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	page := offset / size

	items, err := s.svc.Messages.Conversations(c.Request.Context(), userID, page, size+1)
	if err != nil {
		writeError(c, err)
		return
	}

	response := newConversationResponseList(items)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}

// listConversationMessages godoc
// @Summary		List Conversation Messages
// @Description	Get a paginated list of messages in a specific conversation.
// @Tags			Messages
// @Produce		json
// @Param			id		path		int		true	"Conversation ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[MessageResponse]
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/messages/conversations/{id}/messages [get]
func (s *Server) listConversationMessages(c *gin.Context) {
	var req idURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeError(c, err)
		return
	}

	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	page := offset / size

	items, err := s.svc.Messages.Messages(c.Request.Context(), userID, req.ID, page, size+1)
	if err != nil {
		writeError(c, err)
		return
	}

	response := newMessageResponseList(items)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}

// sendMessageToUser godoc
// @Summary		Send Message to User
// @Description	Start a new conversation or send a message to a user by their ID.
// @Tags			Messages
// @Accept			json
// @Produce		json
// @Param			id		path		int64					true	"Recipient User ID"
// @Param			request	body			sendMessageRequest	true	"Message content"
// @Success		201		{object}	MessageResponse
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/messages/users/{id}/messages [post]
func (s *Server) sendMessageToUser(c *gin.Context) {
	senderID, ok := requireUserID(c)
	if !ok {
		return
	}

	var reqID idURIRequest
	if err := c.ShouldBindUri(&reqID); err != nil {
		writeError(c, err)
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	item, participants, err := s.svc.Messages.SendToUser(c.Request.Context(), senderID, reqID.ID, req.Content)
	if err != nil {
		writeError(c, err)
		return
	}

	resp := newMessageResponse(item)
	s.sendDirectMessageWS(participants, resp)
	c.JSON(http.StatusCreated, resp)
}

// sendMessageToConversation godoc
// @Summary		Send Message to Conversation
// @Description	Send a message to an existing conversation thread.
// @Tags			Messages
// @Accept			json
// @Produce		json
// @Param			id		path		int64					true	"Conversation ID"
// @Param			request	body			sendMessageRequest	true	"Message content"
// @Success		201		{object}	MessageResponse
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/messages/conversations/{id}/messages [post]
func (s *Server) sendMessageToConversation(c *gin.Context) {
	senderID, ok := requireUserID(c)
	if !ok {
		return
	}

	var reqID idURIRequest
	if err := c.ShouldBindUri(&reqID); err != nil {
		writeError(c, err)
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	item, participants, err := s.svc.Messages.SendToConversation(c.Request.Context(), senderID, reqID.ID, req.Content)
	if err != nil {
		writeError(c, err)
		return
	}

	resp := newMessageResponse(item)
	s.sendDirectMessageWS(participants, resp)
	c.JSON(http.StatusCreated, resp)
}

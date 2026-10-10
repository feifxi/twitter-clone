package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	wsWriteWait      = 10 * time.Second
	wsPongWait       = 60 * time.Second
	wsPingPeriod     = 50 * time.Second
	wsMaxMessageSize = 2048
)

type chatWSClient struct {
	userID int64
	conn   *websocket.Conn
	send   chan []byte
}

type wsEnvelope struct {
	Type           string                 `json:"type"`
	ConversationID *int64                 `json:"conversationId,omitempty"`
	Message        any                    `json:"message,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// streamMessagesWS godoc
// @Summary		Stream Messages (WebSocket)
// @Description	Establish a WebSocket connection to receive real-time direct messages. Supports authentication via 'access_token' query param or 'Authorization' header.
// @Tags			Messages
// @Param			access_token	query	string	false	"JWT Access Token (if not using Authorization header)"
// @Success		101				{string}	string	"Switching Protocols"
// @Security		BearerAuth
// @Router			/messages/ws [get]
func (s *Server) streamMessagesWS(c *gin.Context) {
	userID, ok := s.wsUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized websocket connection"})
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			for _, allowed := range s.cfg.AllowedOrigins {
				if allowed == "*" || origin == allowed {
					return true
				}
			}
			return false
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Warn().Err(err).Msg("websocket upgrade failed")
		return
	}

	client := &chatWSClient{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 64),
	}
	s.registerWSClient(client)
	defer s.unregisterWSClient(client)

	if payload, err := json.Marshal(wsEnvelope{
		Type: "connected",
		Data: map[string]interface{}{"status": "ok"},
	}); err == nil {
		select {
		case client.send <- payload:
		default:
		}
	}

	go client.writeLoop()
	client.readLoop()
}

func (s *Server) wsUserID(c *gin.Context) (int64, bool) {
	accessToken := strings.TrimSpace(c.Query("access_token"))
	if accessToken == "" {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			accessToken = strings.TrimSpace(authHeader[7:])
		}
	}
	if accessToken == "" {
		return 0, false
	}

	payload, err := s.tokens.VerifyToken(accessToken)
	if err != nil {
		return 0, false
	}
	return payload.UserID, true
}

func (s *Server) registerWSClient(client *chatWSClient) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	if s.wsClients[client.userID] == nil {
		s.wsClients[client.userID] = make(map[*chatWSClient]struct{})
	}
	s.wsClients[client.userID][client] = struct{}{}
}

func (s *Server) unregisterWSClient(client *chatWSClient) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	if userClients, ok := s.wsClients[client.userID]; ok {
		delete(userClients, client)
		if len(userClients) == 0 {
			delete(s.wsClients, client.userID)
		}
	}
	close(client.send)
}

func (s *Server) sendDirectMessageWS(userIDs []int64, message MessageResponse) {
	payload, err := json.Marshal(wsEnvelope{
		Type:           "dm.message",
		ConversationID: &message.ConversationID,
		Message:        message,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal dm websocket payload")
		return
	}

	for _, userID := range userIDs {
		s.wsMu.RLock()
		for client := range s.wsClients[userID] {
			select {
			case client.send <- payload:
			default:
				log.Warn().Int64("user_id", userID).Msg("dropped dm websocket event due to full client buffer")
			}
		}
		s.wsMu.RUnlock()
	}
}

func (client *chatWSClient) readLoop() {
	defer client.conn.Close()

	client.conn.SetReadLimit(wsMaxMessageSize)
	_ = client.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (client *chatWSClient) writeLoop() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		_ = client.conn.Close()
	}()

	for {
		select {
		case payload, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

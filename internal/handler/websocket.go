package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/psds-microservice/data-channel-service/internal/service"
	"gorm.io/datatypes"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WebSocketHandler struct {
	Hub *service.DataHub
	Svc *service.DataService
}

func NewWebSocketHandler(hub *service.DataHub, svc *service.DataService) *WebSocketHandler {
	return &WebSocketHandler{Hub: hub, Svc: svc}
}

func (h *WebSocketHandler) ServeWS(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := h.Hub.Register(sessionID, userID, conn)
	defer h.Hub.Unregister(sessionID, userID)

	go client.WritePump()
	client.ReadPump(h.Hub, func(sessionID, userID uuid.UUID, payload []byte) {
		_ = h.Svc.AppendMessage(sessionID, userID, "data", datatypes.JSON(payload))
	})
}

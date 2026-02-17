package handler

import (
	"net/http"
	"strconv"

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

type DataHandler struct {
	Hub *service.DataHub
	Svc *service.DataService
}

func NewDataHandler(hub *service.DataHub, svc *service.DataService) *DataHandler {
	return &DataHandler{Hub: hub, Svc: svc}
}

func (h *DataHandler) ServeWS(c *gin.Context) {
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

func (h *DataHandler) GetHistory(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.Svc.GetHistory(sessionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *DataHandler) UploadFile(c *gin.Context) {
	sessionID, err := uuid.Parse(c.PostForm("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}
	userID, err := uuid.Parse(c.PostForm("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, err := h.Svc.SaveFile(sessionID, userID, file.Filename, file.Header.Get("Content-Type"), file.Size, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       f.ID,
		"filename": f.Filename,
		"url":      "/data/file/" + f.ID.String(),
	})
}

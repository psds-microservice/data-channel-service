package service

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type DataHub struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]map[uuid.UUID]*DataConn
}

type DataConn struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	conn      *websocket.Conn
	send      chan []byte
	sendOnce  sync.Once
}

func NewDataHub() *DataHub {
	return &DataHub{
		sessions: make(map[uuid.UUID]map[uuid.UUID]*DataConn),
	}
}

func (c *DataConn) closeSend() { c.sendOnce.Do(func() { close(c.send) }) }

func (h *DataHub) Register(sessionID, userID uuid.UUID, conn *websocket.Conn) *DataConn {
	h.mu.Lock()
	if h.sessions[sessionID] == nil {
		h.sessions[sessionID] = make(map[uuid.UUID]*DataConn)
	}
	if old, ok := h.sessions[sessionID][userID]; ok {
		old.closeSend()
		delete(h.sessions[sessionID], userID)
	}
	c := &DataConn{SessionID: sessionID, UserID: userID, conn: conn, send: make(chan []byte, 256)}
	h.sessions[sessionID][userID] = c
	h.mu.Unlock()
	return c
}

func (h *DataHub) Unregister(sessionID, userID uuid.UUID) {
	h.mu.Lock()
	if m := h.sessions[sessionID]; m != nil {
		if c, ok := m[userID]; ok {
			c.closeSend()
			delete(m, userID)
		}
		if len(m) == 0 {
			delete(h.sessions, sessionID)
		}
	}
	h.mu.Unlock()
}

func (h *DataHub) Broadcast(sessionID uuid.UUID, msg []byte, excludeUserID *uuid.UUID) {
	h.mu.RLock()
	m, ok := h.sessions[sessionID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	copy := make(map[uuid.UUID]*DataConn, len(m))
	for k, v := range m {
		copy[k] = v
	}
	h.mu.RUnlock()
	for uid, c := range copy {
		if excludeUserID != nil && uid == *excludeUserID {
			continue
		}
		if c != nil {
			select {
			case c.send <- msg:
			default:
			}
		}
	}
}

func (c *DataConn) WritePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *DataConn) ReadPump(hub *DataHub, persist func(sessionID, userID uuid.UUID, payload []byte)) {
	defer c.conn.Close()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		hub.Broadcast(c.SessionID, message, &c.UserID)
		if persist != nil {
			persist(c.SessionID, c.UserID, message)
		}
	}
}

package main

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/websocket/v2"
)

// wsEvent is the envelope sent over the websocket in both directions.
// Kinds: "message" (chat text), "location" (lat/lng update), "sos"
// (trigger broadcast to everyone connected).
type wsEvent struct {
	Kind      string   `json:"kind"`
	UserID    int64    `json:"user_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	Body      string   `json:"body,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// hub fans out events to whichever of the (at most 3) family members
// are currently connected. At this user count a mutex-guarded map is
// plenty -- no need for a more elaborate pub/sub layer.
type hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *hub) add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
}

func (h *hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

func (h *hub) broadcast(event wsEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
}

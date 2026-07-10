package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
)

type geochatHandlers struct {
	pool *pgxpool.Pool
	hub  *hub
	sos  *sosService
}

// handleWS is the single websocket endpoint used for both chat
// messages and location updates. SOS can also be sent as a "sos" event
// over this socket, but also has a plain REST fallback below (POST
// /api/geochat/sos) in case a client's websocket connection is the
// thing that's broken.
func (h *geochatHandlers) handleWS() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		user, ok := c.Locals("user").(*auth.User)
		if !ok || user == nil {
			_ = c.Close()
			return
		}

		h.hub.add(c)
		defer h.hub.remove(c)

		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}

			var in wsEvent
			if err := json.Unmarshal(raw, &in); err != nil {
				continue
			}

			switch in.Kind {
			case "message":
				h.handleChatMessage(user, in.Body)
			case "location":
				h.handleLocationUpdate(user, in.Latitude, in.Longitude)
			case "sos":
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				h.sos.trigger(ctx, user.ID, user.Username, in.Body, in.Latitude, in.Longitude)
				cancel()
			}
		}
	})
}

// handleChatMessage persists the message then re-broadcasts it with the
// server-assigned id/timestamp so all connected clients (including the
// sender) render one consistent copy.
func (h *geochatHandlers) handleChatMessage(user *auth.User, body string) {
	if body == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var createdAt time.Time
	err := h.pool.QueryRow(ctx, `
		INSERT INTO geochat.messages (user_id, body, kind)
		VALUES ($1, $2, 'text')
		RETURNING created_at
	`, user.ID, body).Scan(&createdAt)
	if err != nil {
		return
	}

	h.hub.broadcast(wsEvent{
		Kind:      "message",
		UserID:    user.ID,
		Username:  user.Username,
		Body:      body,
		CreatedAt: createdAt.Format(time.RFC3339),
	})
}

// handleLocationUpdate upserts the sender's latest known location and
// re-broadcasts it -- this app tracks "where is everyone right now",
// not a location history.
func (h *geochatHandlers) handleLocationUpdate(user *auth.User, lat, lng *float64) {
	if lat == nil || lng == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.pool.Exec(ctx, `
		INSERT INTO geochat.locations (user_id, latitude, longitude, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET latitude = $2, longitude = $3, updated_at = now()
	`, user.ID, *lat, *lng)
	if err != nil {
		return
	}

	h.hub.broadcast(wsEvent{
		Kind:      "location",
		UserID:    user.ID,
		Username:  user.Username,
		Latitude:  lat,
		Longitude: lng,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *geochatHandlers) listMessages(c *fiber.Ctx) error {
	rows, err := h.pool.Query(c.Context(), `
		SELECT m.id, m.user_id, u.username, m.body, m.kind, m.created_at
		FROM geochat.messages m
		JOIN auth.users u ON u.id = m.user_id
		ORDER BY m.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type message struct {
		ID        int64  `json:"id"`
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		Body      string `json:"body"`
		Kind      string `json:"kind"`
		CreatedAt string `json:"created_at"`
	}

	out := []message{}
	for rows.Next() {
		var m message
		var createdAt time.Time
		if err := rows.Scan(&m.ID, &m.UserID, &m.Username, &m.Body, &m.Kind, &createdAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		out = append(out, m)
	}
	return c.JSON(out)
}

func (h *geochatHandlers) latestLocations(c *fiber.Ctx) error {
	rows, err := h.pool.Query(c.Context(), `
		SELECT u.id, u.username, l.latitude, l.longitude, l.updated_at
		FROM geochat.locations l
		JOIN auth.users u ON u.id = l.user_id
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type loc struct {
		UserID    int64   `json:"user_id"`
		Username  string  `json:"username"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		UpdatedAt string  `json:"updated_at"`
	}

	out := []loc{}
	for rows.Next() {
		var l loc
		var updatedAt time.Time
		if err := rows.Scan(&l.UserID, &l.Username, &l.Latitude, &l.Longitude, &updatedAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		l.UpdatedAt = updatedAt.Format(time.RFC3339)
		out = append(out, l)
	}
	return c.JSON(out)
}

// triggerSOSHTTP is the REST fallback for SOS: it works even from a
// client whose websocket connection dropped, as long as it can still
// reach HAProxy at all.
func (h *geochatHandlers) triggerSOSHTTP(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	var body struct {
		Message   string   `json:"message"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Message == "" {
		body.Message = "I need help"
	}

	telegramOK := h.sos.trigger(c.Context(), user.ID, user.Username, body.Message, body.Latitude, body.Longitude)

	return c.JSON(fiber.Map{"ok": true, "telegram_ok": telegramOK})
}

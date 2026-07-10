package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sosLogger appends one line per SOS trigger to a plain file, independent
// of the Postgres row also written for the same event. The doc calls
// for this specifically so a trigger can be audited even if the
// database write path is what's broken.
type sosLogger struct {
	mu   sync.Mutex
	path string
}

func newSOSLogger(path string) *sosLogger {
	return &sosLogger{path: path}
}

func (l *sosLogger) append(username string, lat, lng *float64, telegramOK bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	coords := "unknown"
	if lat != nil && lng != nil {
		coords = fmt.Sprintf("%.6f,%.6f", *lat, *lng)
	}

	line := fmt.Sprintf("%s\tuser=%s\tcoords=%s\ttelegram_ok=%t\n",
		time.Now().UTC().Format(time.RFC3339), username, coords, telegramOK)
	_, err = f.WriteString(line)
	return err
}

type sosService struct {
	pool     *pgxpool.Pool
	hub      *hub
	telegram *telegramNotifier
	fileLog  *sosLogger
}

// trigger fans an SOS out across every channel this stack has: the
// live websocket broadcast (fast, but only reaches connected clients on
// a working VPS), the Telegram fallback (reaches phones even if the
// VPS is down), and two independent audit trails (DB row + append-only
// file).
func (s *sosService) trigger(ctx context.Context, userID int64, username, message string, lat, lng *float64) (telegramOK bool) {
	s.hub.broadcast(wsEvent{
		Kind:      "sos",
		UserID:    userID,
		Username:  username,
		Body:      message,
		Latitude:  lat,
		Longitude: lng,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	telegramOK = s.telegram.notifySOS(ctx, username, message, lat, lng)

	_, _ = s.pool.Exec(ctx, `
		INSERT INTO geochat.sos_events (user_id, latitude, longitude, message, telegram_ok)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, lat, lng, message, telegramOK)

	_ = s.fileLog.append(username, lat, lng, telegramOK)

	return telegramOK
}

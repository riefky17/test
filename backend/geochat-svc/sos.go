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

func (l *sosLogger) append(username string, lat, lng *float64, recipientsOnline int) error {
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

	line := fmt.Sprintf("%s\tuser=%s\tcoords=%s\trecipients_online=%d\n",
		time.Now().UTC().Format(time.RFC3339), username, coords, recipientsOnline)
	_, err = f.WriteString(line)
	return err
}

type sosService struct {
	pool    *pgxpool.Pool
	hub     *hub
	fileLog *sosLogger
}

// trigger is Geochat's whole SOS story: everything runs through this
// same web messenger, there's no separate third-party bot to fall back
// to. It broadcasts over the live websocket (the only delivery path
// there is) and writes two independent audit trails -- a Postgres row
// and an append-only file -- recording exactly how many other family
// members were actually online to see it at the moment it fired, so a
// trigger can be reviewed honestly afterwards even if nobody was
// connected to receive it live.
func (s *sosService) trigger(ctx context.Context, userID int64, username, message string, lat, lng *float64) (recipientsOnline int) {
	recipientsOnline = s.hub.onlineOthers(userID)

	s.hub.broadcast(wsEvent{
		Kind:      "sos",
		UserID:    userID,
		Username:  username,
		Body:      message,
		Latitude:  lat,
		Longitude: lng,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	_, _ = s.pool.Exec(ctx, `
		INSERT INTO geochat.sos_events (user_id, latitude, longitude, message, recipients_online)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, lat, lng, message, recipientsOnline)

	_ = s.fileLog.append(username, lat, lng, recipientsOnline)

	return recipientsOnline
}

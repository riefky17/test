package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// telegramNotifier is the SOS fallback channel. It does not touch the
// VPS at all -- it's an outbound HTTPS call to Telegram's API, so it
// keeps working even if geochat-svc's own websocket/broadcast path is
// down. That's the whole point: it's the one thing in this stack that
// must not share a single point of failure with the app server.
type telegramNotifier struct {
	botToken string
	chatIDs  []string
	client   *http.Client
}

func newTelegramNotifier(botToken string, chatIDs []string) *telegramNotifier {
	return &telegramNotifier{
		botToken: botToken,
		chatIDs:  chatIDs,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *telegramNotifier) enabled() bool {
	return t.botToken != "" && len(t.chatIDs) > 0
}

// notifySOS fires sendMessage + sendLocation (when coordinates are
// known) to every configured chat ID. It returns true only if every
// call to every recipient succeeded, since a partial SOS delivery is
// still a failure worth logging as such.
func (t *telegramNotifier) notifySOS(ctx context.Context, senderName, message string, lat, lng *float64) bool {
	if !t.enabled() {
		return false
	}

	allOK := true
	for _, chatID := range t.chatIDs {
		if err := t.sendMessage(ctx, chatID, fmt.Sprintf("🆘 SOS from %s: %s", senderName, message)); err != nil {
			allOK = false
			continue
		}
		if lat != nil && lng != nil {
			if err := t.sendLocation(ctx, chatID, *lat, *lng); err != nil {
				allOK = false
			}
		}
	}
	return allOK
}

func (t *telegramNotifier) sendMessage(ctx context.Context, chatID, text string) error {
	return t.call(ctx, "sendMessage", url.Values{
		"chat_id": {chatID},
		"text":    {text},
	})
}

func (t *telegramNotifier) sendLocation(ctx context.Context, chatID string, lat, lng float64) error {
	return t.call(ctx, "sendLocation", url.Values{
		"chat_id":   {chatID},
		"latitude":  {strconv.FormatFloat(lat, 'f', -1, 64)},
		"longitude": {strconv.FormatFloat(lng, 'f', -1, 64)},
	})
}

func (t *telegramNotifier) call(ctx context.Context, method string, params url.Values) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", t.botToken, method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	req.URL.RawQuery = params.Encode()

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var body struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return fmt.Errorf("telegram %s failed: %s (%s)", method, resp.Status, body.Description)
	}
	return nil
}

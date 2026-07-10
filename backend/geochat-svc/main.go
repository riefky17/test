// geochat-svc is the family chat + location + SOS backend. It listens
// on 127.0.0.1:3002. This is the one service where downtime actually
// matters in an emergency, which is why SOS triggers also fan out to
// Telegram (see telegram.go / sos.go) instead of depending solely on
// this process and this VPS being up.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
	"github.com/riefky17/family-app-suite/backend/internal/config"
	"github.com/riefky17/family-app-suite/backend/internal/db"
	"github.com/riefky17/family-app-suite/backend/internal/spa"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("127.0.0.1:3002")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if !newTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatIDs).enabled() {
		log.Println("WARNING: TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_IDS not set -- SOS has no fallback channel, only the in-app broadcast")
	}

	sosLogPath := os.Getenv("SOS_LOG_PATH")
	if sosLogPath == "" {
		sosLogPath = "/var/log/geochat/sos.log"
	}

	authStore := auth.NewStore(pool)
	h := newHub()
	handlers := &geochatHandlers{
		pool: pool,
		hub:  h,
		sos: &sosService{
			pool:     pool,
			hub:      h,
			telegram: newTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatIDs),
			fileLog:  newSOSLogger(sosLogPath),
		},
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())
	app.Use(recover.New())

	app.Post("/api/auth/login", auth.LoginHandler(authStore, auth.CookieOptions{
		Domain: cfg.CookieDomain,
		Secure: cfg.CookieSecure,
	}))
	app.Post("/api/auth/logout", auth.LogoutHandler(authStore))
	app.Get("/api/auth/me", auth.Middleware(authStore), auth.MeHandler())

	// Auth runs first so the websocket handler can read the user back
	// out of c.Locals("user") -- websocket.Conn has no request context
	// of its own to re-validate the session cookie with.
	app.Use("/api/geochat/ws", auth.Middleware(authStore))
	app.Use("/api/geochat/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/geochat/ws", handlers.handleWS())

	api := app.Group("/api/geochat", auth.Middleware(authStore))
	api.Get("/messages", handlers.listMessages)
	api.Get("/locations/latest", handlers.latestLocations)
	api.Post("/sos", handlers.triggerSOSHTTP)

	spa.Register(app, cfg.StaticDir)

	go func() {
		<-ctx.Done()
		log.Println("geochat-svc: shutting down")
		_ = app.Shutdown()
	}()

	log.Printf("geochat-svc listening on %s", cfg.ListenAddr)
	if err := app.Listen(cfg.ListenAddr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

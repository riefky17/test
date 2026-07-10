// finance-svc is the Finance Tracker backend. It listens on
// 127.0.0.1:3001 and is only reachable through HAProxy -- see
// deploy/haproxy/haproxy.cfg. A crash here never touches Geochat's
// websocket, since the two run as separate systemd units.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
	"github.com/riefky17/family-app-suite/backend/internal/config"
	"github.com/riefky17/family-app-suite/backend/internal/db"
	"github.com/riefky17/family-app-suite/backend/internal/spa"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("127.0.0.1:3001")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	authStore := auth.NewStore(pool)
	svc := &financeService{pool: pool}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())
	app.Use(recover.New())

	app.Post("/api/auth/login", auth.LoginHandler(authStore, auth.CookieOptions{
		Domain: cfg.CookieDomain,
		Secure: cfg.CookieSecure,
	}))
	app.Post("/api/auth/logout", auth.LogoutHandler(authStore))
	app.Get("/api/auth/me", auth.Middleware(authStore), auth.MeHandler())

	api := app.Group("/api/finance", auth.Middleware(authStore))
	api.Get("/categories", svc.listCategories)
	api.Get("/transactions", svc.listTransactions)
	api.Post("/transactions", svc.createTransaction)
	api.Delete("/transactions/:id", svc.deleteTransaction)
	api.Get("/summary", svc.summary)

	spa.Register(app, cfg.StaticDir)

	go func() {
		<-ctx.Done()
		log.Println("finance-svc: shutting down")
		_ = app.Shutdown()
	}()

	log.Printf("finance-svc listening on %s", cfg.ListenAddr)
	if err := app.Listen(cfg.ListenAddr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

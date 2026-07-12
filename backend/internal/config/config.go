// Package config loads process configuration from environment variables.
// All 3 services (finance, geochat, fitness) share this loader so env
// handling stays consistent across binaries.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	// DatabaseURL points at the single shared Postgres instance; each
	// service connects with its own small pool (see internal/db).
	DatabaseURL string
	// ListenAddr is the loopback address the service binds to. HAProxy
	// is the only thing that talks to these ports directly.
	ListenAddr string
	// CookieDomain / CookieSecure control the session cookie set by
	// internal/auth. In production (behind Cloudflare Tunnel) Secure
	// must be true.
	CookieDomain string
	CookieSecure bool
	// StaticDir is the built frontend (frontend/dist) each service
	// serves directly -- HAProxy routes a whole hostname to exactly one
	// backend, so whichever service answers for e.g. geochat.yourdomain.com
	// must also be able to serve the SPA shell for it.
	StaticDir string
}

func Load(defaultAddr string) (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./public"
	}

	cfg := &Config{
		DatabaseURL:  dbURL,
		ListenAddr:   addr,
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),
		CookieSecure: os.Getenv("COOKIE_SECURE") != "false",
		StaticDir:    staticDir,
	}

	return cfg, nil
}

// cmd/seed is a one-off setup script that creates the 3 family
// accounts (Papa, Mami, Echa). There's no self-registration flow --
// this is the only way accounts get created, run once against a fresh
// database.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed \
//	  papa:<password> mami:<password> echa:<password>
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
	"github.com/riefky17/family-app-suite/backend/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: seed <username:password> [<username:password> ...]")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	for _, arg := range os.Args[1:] {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			log.Fatalf("invalid argument %q, expected username:password", arg)
		}
		username, password := parts[0], parts[1]

		hash, err := auth.HashPassword(password)
		if err != nil {
			log.Fatalf("hash password for %s: %v", username, err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO auth.users (username, display_name, password_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		`, username, capitalize(username), hash)
		if err != nil {
			log.Fatalf("insert user %s: %v", username, err)
		}

		fmt.Printf("seeded user %q\n", username)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

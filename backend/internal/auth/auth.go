// Package auth is the single shared login/session package compiled into
// all 3 service binaries (finance-svc, geochat-svc, fitness-svc), so
// password hashing and session validation logic lives in one place
// instead of being duplicated 3 times. Sessions are stored in the
// `auth.sessions` table in the shared Postgres instance and checked on
// every request via cookie -- deliberately simple, since there are only
// ever 3 known users and no third-party identity provider is needed.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const SessionTTL = 30 * 24 * time.Hour

var ErrInvalidCredentials = errors.New("invalid username or password")

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Authenticate checks username/password against auth.users and returns
// the matched user. It always runs the bcrypt comparison work even on a
// missing username, using a fixed dummy hash, so failed lookups don't
// leak timing information about which usernames exist.
func (s *Store) Authenticate(ctx context.Context, username, password string) (*User, error) {
	var (
		id           int64
		passwordHash string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, password_hash FROM auth.users WHERE username = $1`,
		username,
	).Scan(&id, &passwordHash)

	const dummyHash = "$2a$10$C6UzMDM.H6dfI/f/IKcEeO....gOgHZQnHnkgSJ1s.KxTv5N4Vsxa"
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &User{ID: id, Username: username}, nil
}

func (s *Store) CreateSession(ctx context.Context, userID int64) (token string, expiresAt time.Time, err error) {
	token, err = randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(SessionTTL)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO auth.sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *Store) ValidateSession(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}

	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.username
		 FROM auth.sessions s
		 JOIN auth.users u ON u.id = s.user_id
		 WHERE s.token = $1 AND s.expires_at > now()`,
		token,
	).Scan(&user.ID, &user.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth.sessions WHERE token = $1`, token)
	return err
}

func randomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

const CookieName = "family_session"

// Middleware rejects any request without a valid session cookie and
// stashes the authenticated user in fiber.Locals("user") for handlers
// to read.
func Middleware(store *Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CookieName)
		user, err := store.ValidateSession(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals("user", user)
		return c.Next()
	}
}

func CurrentUser(c *fiber.Ctx) *User {
	user, _ := c.Locals("user").(*User)
	return user
}

type CookieOptions struct {
	Domain string
	Secure bool
}

// LoginHandler returns a POST handler that authenticates a username and
// password and sets the session cookie. Shared by all 3 services so the
// login form on the frontend can point at any of them.
func LoginHandler(store *Store, opts CookieOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		user, err := store.Authenticate(c.Context(), body.Username, body.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
		}

		token, expiresAt, err := store.CreateSession(c.Context(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create session"})
		}

		c.Cookie(&fiber.Cookie{
			Name:     CookieName,
			Value:    token,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   opts.Secure,
			SameSite: "Lax",
			Domain:   opts.Domain,
			Path:     "/",
		})

		return c.JSON(fiber.Map{"user": user})
	}
}

func LogoutHandler(store *Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CookieName)
		if token != "" {
			_ = store.DeleteSession(c.Context(), token)
		}
		c.ClearCookie(CookieName)
		return c.JSON(fiber.Map{"ok": true})
	}
}

func MeHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"user": CurrentUser(c)})
	}
}

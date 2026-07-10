// Package spa serves the shared frontend build (frontend/dist, copied
// to each service's STATIC_DIR at deploy time). Every one of the 3 Go
// binaries mounts this the same way, since HAProxy sends each hostname
// to exactly one backend and that backend is the only thing able to
// answer with HTML for it.
package spa

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// Register must be called after all /api routes are registered: it
// installs a catch-all that serves index.html for any unmatched path so
// client-side routing (react-router) works on refresh/deep links.
func Register(app *fiber.App, dir string) {
	app.Use(filesystem.New(filesystem.Config{
		Root:   http.Dir(dir),
		Index:  "index.html",
		Browse: false,
	}))

	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile(dir + "/index.html")
	})
}

package admin

import "github.com/gofiber/fiber/v3"

// Auth menolak request tanpa cookie sesi admin yang valid.
// Token Bearer user (header Authorization) sengaja tidak diterima di sini.
func Auth(cfg Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := parseSessionToken(c.Cookies(CookieName), cfg.Secret); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "admin login required")
		}
		return c.Next()
	}
}

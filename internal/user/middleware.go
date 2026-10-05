package user

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const userIDKey = "userID"

// Auth menolak request tanpa header "Authorization: Bearer <token>" yang valid,
// lalu menyimpan ID user di c.Locals(userIDKey).
func Auth(secret string) func(fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		token, ok := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}

		id, err := ParseToken(token, secret)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(userIDKey, id)

		return c.Next()
	}
}

// UserID mengambil ID user yang disimpan Auth.
func UserID(c fiber.Ctx) int64 {
	id, _ := c.Locals(userIDKey).(int64)
	return id
}

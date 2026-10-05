package main

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

// reportError tulis ke log. Body request sengaja tidak ikut (ada password).
func reportError(c fiber.Ctx, msg, stack string) {
	slog.Error("request failed", "method", c.Method(), "path", c.Path(), "err", msg, "stack", stack)
}

// errorHandler: *fiber.Error (4xx dll) sengaja dilempar handler, cukup balas.
// Error lain = kegagalan server: log, client dapat 500 generik.
func errorHandler(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}

	reportError(c, err.Error(), "")
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

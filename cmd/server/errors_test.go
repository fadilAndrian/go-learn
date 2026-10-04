package main

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

func TestErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Use(recover.New(recover.Config{
		PanicHandler: func(c fiber.Ctx, r any) error {
			reportError(c, "panic", "")
			return fiber.ErrInternalServerError
		},
	}))
	app.Get("/4xx", func(fiber.Ctx) error { return fiber.NewError(fiber.StatusBadRequest, "bad") })
	app.Get("/err", func(fiber.Ctx) error { return errors.New("db down") })
	app.Get("/panic", func(fiber.Ctx) error { panic("x") })

	for path, want := range map[string]int{"/4xx": 400, "/err": 500, "/panic": 500} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil || resp.StatusCode != want {
			t.Fatalf("%s: got %v %v, want %d", path, resp, err, want)
		}
	}
}

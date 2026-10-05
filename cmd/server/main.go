package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/fadilAndrian/go-learn/internal/merchant"
	"github.com/fadilAndrian/go-learn/internal/user"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// fatal log error lalu keluar, pengganti log.Fatal
func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}

func main() {
	// log JSON ke terminal + logs/app.log, setara laravel.log
	// ponytail: tanpa rotasi, file tumbuh terus. Pakai logrotate / lumberjack kalau sudah besar.
	out := io.Writer(os.Stdout)
	if err := os.MkdirAll("logs", 0o755); err == nil {
		if f, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			out = io.MultiWriter(os.Stdout, f)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil)))

	// baca env
	if err := godotenv.Load(); err != nil {
		fatal("load .env", err)
	}

	// karena nanti di authentikasi butuh JWT,
	// jadi ini untuk setup secret key nya
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		fatal("JWT_SECRET is not set", nil)
	}

	// connection.go hehe
	db, err := pgxpool.New(context.Background(), os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		fatal("connect db", err)
	}
	defer db.Close()

	// inisiasi http pakai fiber
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})

	// panic ditangkap, dilog + notif, client dapat 500
	app.Use(recover.New(recover.Config{
		PanicHandler: func(c fiber.Ctx, r any) error {
			reportError(c, fmt.Sprint("panic: ", r), string(debug.Stack()))
			return fiber.ErrInternalServerError
		},
	}))

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo, jwtSecret)
	userHandler := user.NewHandler(userService)

	app.Post("/register", userHandler.Register)
	app.Post("/login", userHandler.Login)
	app.Get("/me", user.Auth(jwtSecret), userHandler.Me)

	merchantHandler := merchant.NewHandler(merchant.NewService(merchant.NewRepository(db)))

	app.Post("/merchants", merchantHandler.Register)
	app.Get("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Detail)
	app.Put("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Update)
	app.Delete("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Delete)

	// arahin ke port 3000
	if err := app.Listen(":3000"); err != nil {
		fatal("listen", err)
	}
}

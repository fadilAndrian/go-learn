package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bagian yang sama untuk semua subcommand: logger, DB, Fiber, listen.

// fatal log error lalu keluar, pengganti log.Fatal
func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}

// log JSON ke terminal + logs/app.log, setara laravel.log
// ponytail: tanpa rotasi, file tumbuh terus. Pakai logrotate / lumberjack kalau sudah besar.
func setupLogger() {
	out := io.Writer(os.Stdout)
	if err := os.MkdirAll("logs", 0o755); err == nil {
		if f, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			out = io.MultiWriter(os.Stdout, f)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil)))
}

// connection.go hehe
func openDB(cfg Config) *pgxpool.Pool {
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fatal("connect db", err)
	}
	return db
}

// newApp: Fiber dengan error handler pusat + recover.
// Panic ditangkap, dilog, client dapat 500.
func newApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Use(recover.New(recover.Config{
		PanicHandler: func(c fiber.Ctx, r any) error {
			reportError(c, fmt.Sprint("panic: ", r), string(debug.Stack()))
			return fiber.ErrInternalServerError
		},
	}))
	return app
}

// listen jalan sampai Ctrl+C / SIGTERM, lalu menunggu request yang sedang
// diproses selesai (graceful shutdown) sebelum kembali.
func listen(app *fiber.App, name, port string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("listening", "server", name, "port", port)
	if err := app.Listen(":"+port, fiber.ListenConfig{GracefulContext: ctx}); err != nil {
		fatal("listen", err)
	}
	slog.Info("stopped", "server", name)
}

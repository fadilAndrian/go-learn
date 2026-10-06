package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/fadilAndrian/go-learn/internal/apilog"
	"github.com/fadilAndrian/go-learn/internal/gateway"
	"github.com/fadilAndrian/go-learn/internal/merchant"
	"github.com/fadilAndrian/go-learn/internal/transaction"
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

	// admin-api
	app.Post("/register", userHandler.Register)
	app.Post("/login", userHandler.Login)
	app.Get("/me", user.Auth(jwtSecret), userHandler.Me)

	// payment gateway: PG_BASE_URL kosong + PG_FAKE=true → pakai PG palsu bawaan di /fake-pg
	pgBase := os.Getenv("PG_BASE_URL")
	if pgBase == "" {
		pgBase = "http://localhost:3000/fake-pg"
	}
	webhookSecret := os.Getenv("PG_WEBHOOK_SECRET")
	if os.Getenv("PG_FAKE") == "true" {
		gateway.RegisterFake(app, "http://localhost:3000/webhooks/qris/v1.0/qr/qr-mpm-notify", webhookSecret)
	}

	logs := apilog.NewService(apilog.NewRepository(db))
	gw := gateway.NewClient(pgBase, os.Getenv("PG_PARTNER_ID"))

	trxSvc := transaction.NewService(transaction.NewRepository(db), gw, logs)
	trxHandler := transaction.NewHandler(trxSvc, userService, logs)

	// scheduler cek status pending ke PG; PG_CHECK_INTERVAL default 1m, "0" = mati
	every := time.Minute
	if v := os.Getenv("PG_CHECK_INTERVAL"); v != "" {
		if every, err = time.ParseDuration(v); err != nil {
			fatal("PG_CHECK_INTERVAL", err)
		}
	}
	if every > 0 {
		// ponytail: ctx Background, goroutine mati bersama proses. Pakai signal.NotifyContext + Fiber GracefulContext kalau butuh graceful shutdown.
		go trxSvc.RunChecker(context.Background(), every)
	}

	// webhook dari PG: publik, auth lewat X-SIGNATURE
	app.Post("/webhooks/qris/v1.0/qr/qr-mpm-notify", trxHandler.Notify(webhookSecret))

	trx := app.Group("/transactions", user.Auth(jwtSecret))
	trx.Post("/", trxHandler.CreateQRIS) // inbound create → otomatis outbound ke PG
	// admin-api
	trx.Get("/", trxHandler.List)
	trx.Get("/:id", trxHandler.Get)
	trx.Get("/:id/logs", trxHandler.Logs)
	trx.Post("/:id/check", trxHandler.Check)
	trx.Post("/:id/refund", trxHandler.Refund)

	// admin-api
	app.Get("/logs/:logId", user.Auth(jwtSecret), trxHandler.LogDetail)

	merchantHandler := merchant.NewHandler(merchant.NewService(merchant.NewRepository(db)))
	// admin-api
	app.Post("/merchants", merchantHandler.Register)
	app.Get("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Detail)
	app.Put("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Update)
	app.Delete("/merchants/:id", user.Auth(jwtSecret), merchantHandler.Delete)

	// arahin ke port 3000
	if err := app.Listen(":3000"); err != nil {
		fatal("listen", err)
	}
}

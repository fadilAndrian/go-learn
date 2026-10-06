package main

import (
	"github.com/fadilAndrian/go-learn/internal/apilog"
	"github.com/fadilAndrian/go-learn/internal/gateway"
	"github.com/fadilAndrian/go-learn/internal/transaction"
	"github.com/fadilAndrian/go-learn/internal/user"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// serve-api: dipanggil aplikasi user/POS (auth Bearer token) dan PG (webhook).
func runServeAPI(cfg Config) {
	db := openDB(cfg)
	defer db.Close()

	app := newApp()
	apiRoutes(app, cfg, db)
	listen(app, "api", cfg.APIPort)
}

func apiRoutes(app *fiber.App, cfg Config, db *pgxpool.Pool) {
	userService := user.NewService(user.NewRepository(db), cfg.JWTSecret)
	userHandler := user.NewHandler(userService)

	// admin-api
	app.Post("/register", userHandler.Register)
	app.Post("/login", userHandler.Login)
	app.Get("/me", user.Auth(cfg.JWTSecret), userHandler.Me)

	// payment gateway: PG_BASE_URL kosong + PG_FAKE=true → pakai PG palsu bawaan di /fake-pg
	self := "http://localhost:" + cfg.APIPort
	pgBase := cfg.PGBaseURL
	if pgBase == "" {
		pgBase = self + "/fake-pg"
	}
	if cfg.PGFake {
		gateway.RegisterFake(app, self+"/webhooks/qris/v1.0/qr/qr-mpm-notify", cfg.PGWebhookSecret)
	}

	logs := apilog.NewService(apilog.NewRepository(db))
	gw := gateway.NewClient(pgBase, cfg.PGPartnerID)
	trxHandler := transaction.NewHandler(transaction.NewService(transaction.NewRepository(db), gw, logs), userService, logs)

	// webhook dari PG: publik, auth lewat X-SIGNATURE
	app.Post("/webhooks/qris/v1.0/qr/qr-mpm-notify", trxHandler.Notify(cfg.PGWebhookSecret))

	trx := app.Group("/transactions", user.Auth(cfg.JWTSecret))
	trx.Post("/", trxHandler.CreateQRIS) // inbound create → otomatis outbound ke PG
	// admin-api
	trx.Get("/", trxHandler.List)
	trx.Get("/:id", trxHandler.Get)
	trx.Get("/:id/logs", trxHandler.Logs)
	trx.Post("/:id/check", trxHandler.Check)
	trx.Post("/:id/refund", trxHandler.Refund)

	// admin-api
	app.Get("/logs/:logId", user.Auth(cfg.JWTSecret), trxHandler.LogDetail)
}

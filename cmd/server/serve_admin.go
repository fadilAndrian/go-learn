package main

import (
	"github.com/fadilAndrian/go-learn/internal/admin"
	"github.com/fadilAndrian/go-learn/internal/merchant"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// serve-admin: dipanggil dashboard admin. Auth pakai cookie sesi (bukan Bearer
// token user) dan port-nya terpisah, jadi bisa ditutup dari internet.
func runServeAdmin(cfg Config) {
	if cfg.AdminKey == "" {
		fatal("ADMIN_KEY is not set", nil)
	}
	db := openDB(cfg)
	defer db.Close()

	app := newApp()
	adminRoutes(app, cfg, db)
	listen(app, "admin", cfg.AdminPort)
}

func adminRoutes(app *fiber.App, cfg Config, db *pgxpool.Pool) {
	adminCfg := admin.Config{
		Key:          cfg.AdminKey,
		Secret:       cfg.JWTSecret,
		SecureCookie: cfg.AdminCookieSecure,
	}
	adminHandler := admin.NewHandler(adminCfg)
	app.Post("/admin/login", adminHandler.Login)
	app.Post("/admin/logout", adminHandler.Logout)

	merchantHandler := merchant.NewHandler(merchant.NewService(merchant.NewRepository(db)))
	merchants := app.Group("/admin/merchants", admin.Auth(adminCfg))
	merchants.Post("/", merchantHandler.Store)
	merchants.Get("/:id", merchantHandler.Show)
	merchants.Put("/:id", merchantHandler.Update)
	merchants.Delete("/:id", merchantHandler.Destroy)
}

package main

import (
	"os"

	"github.com/joho/godotenv"
)

// Config = semua env yang dibaca aplikasi (≈ folder config/ di Laravel).
// os.Getenv hanya boleh ada di file ini.
type Config struct {
	DatabaseURL string
	JWTSecret   string

	APIPort   string
	AdminPort string

	PGBaseURL       string
	PGPartnerID     string
	PGWebhookSecret string
	PGFake          bool

	AdminKey          string
	AdminCookieSecure bool
}

// loadConfig membaca .env lalu env. Hanya yang dibutuhkan SEMUA subcommand
// yang dicek di sini; yang khusus satu subcommand dicek di serve_<nama>.go.
func loadConfig() Config {
	if err := godotenv.Load(); err != nil {
		fatal("load .env", err)
	}

	cfg := Config{
		DatabaseURL: os.Getenv("GOOSE_DBSTRING"),
		JWTSecret:   os.Getenv("JWT_SECRET"),

		APIPort:   envOr("API_PORT", "3000"),
		AdminPort: envOr("ADMIN_PORT", "3001"),

		PGBaseURL:       os.Getenv("PG_BASE_URL"),
		PGPartnerID:     os.Getenv("PG_PARTNER_ID"),
		PGWebhookSecret: os.Getenv("PG_WEBHOOK_SECRET"),
		PGFake:          os.Getenv("PG_FAKE") == "true",

		AdminKey:          os.Getenv("ADMIN_KEY"),
		AdminCookieSecure: os.Getenv("ADMIN_COOKIE_SECURE") != "false",
	}

	// karena di authentikasi butuh JWT, secret wajib ada
	if cfg.JWTSecret == "" {
		fatal("JWT_SECRET is not set", nil)
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

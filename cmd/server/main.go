package main

import (
	"fmt"
	"os"
)

// Satu binary, satu subcommand per pemanggil (≈ php artisan <command>):
//
//	go run ./cmd/server serve-api     POS/user API + webhook PG   (API_PORT, default 3000)
//	go run ./cmd/server serve-admin   dashboard admin, auth cookie (ADMIN_PORT, default 3001)
//
// Mau tahu sebuah endpoint dipanggil siapa? Buka serve_<nama>.go.
func main() {
	setupLogger()
	cfg := loadConfig()

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve-api":
		runServeAPI(cfg)
	case "serve-admin":
		runServeAdmin(cfg)
	default:
		fmt.Fprintln(os.Stderr, "pakai: server serve-api | serve-admin")
		os.Exit(2)
	}
}

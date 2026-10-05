package main

import (
	"context"
	"log"
	"os"

	"github.com/fadilAndrian/go-learn/internal/apilog"
	"github.com/fadilAndrian/go-learn/internal/transaction"
	"github.com/fadilAndrian/go-learn/internal/user"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// baca env
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	// karena nanti di authentikasi butuh JWT,
	// jadi ini untuk setup secret key nya
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	// connection.go hehe
	db, err := pgxpool.New(context.Background(), os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// inisiasi http pakai fiber
	app := fiber.New()

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo, jwtSecret)
	userHandler := user.NewHandler(userService)

	app.Post("/register", userHandler.Register)
	app.Post("/login", userHandler.Login)
	app.Get("/me", user.Auth(jwtSecret), userHandler.Me)

	trxHandler := transaction.NewHandler(transaction.NewService(transaction.NewRepository(db)), userService, apilog.NewService(apilog.NewRepository(db)))

	trx := app.Group("/transactions", user.Auth(jwtSecret))
	trx.Post("/", trxHandler.Create)
	trx.Get("/", trxHandler.List)
	trx.Get("/:id", trxHandler.Get)
	trx.Get("/:id/logs", trxHandler.Logs)

	app.Get("/logs/:logId", user.Auth(jwtSecret), trxHandler.LogDetail)

	// arahin ke port 3000
	log.Fatal(app.Listen(":3000"))
}

package main

import (
	"log"

	"github.com/joho/godotenv"
	"kriyatec.com/go-api/pkg/admin-service/authentication"
	"kriyatec.com/go-api/pkg/admin-service/entities"
	"kriyatec.com/go-api/pkg/admin-service/info"
	"kriyatec.com/go-api/pkg/shared/database"
	"kriyatec.com/go-api/server"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Server initialization
	app := server.Create()
	//By Default try to connect shared db
	database.Init()
	info.SetupRoutes(app)

	authentication.SetupRoutes(app)
	entities.SetupAllRoutes(app)

	if err := server.Listen(app); err != nil {
		log.Panic(err)
	}
}

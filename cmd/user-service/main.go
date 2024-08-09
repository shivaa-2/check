package main

import (
	"log"

	"github.com/joho/godotenv"
	"kriyatec.com/go-api/server"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Server initialization
	app := server.Create()

	// Migrations
	//database.DB.AutoMigrate(&books.Book{})

	// Api routes
	//api.Setup(app)

	if err := server.Listen(app); err != nil {
		log.Panic(err)
	}
}

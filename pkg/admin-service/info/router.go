package info

import (
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	info := app.Group("/static")
	info.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Static Info APIs")
	})
	info.Get("/:id", getDocByIdHandler)
}

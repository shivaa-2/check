package authentication

import (
	"github.com/gofiber/fiber/v2"

	"kriyatec.com/go-api/pkg/shared/helper"
)

func SetupRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	auth := app.Group("/auth")
	auth.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Auth APIs")
	})
	auth.Post("/login", LoginHandler)
	auth.Post("/shop/login", EmpLoginHandler)
	auth.Post("/otp", OTPValidateHandler)
	auth.Post("/register", RegistrationHandler)
	auth.Get("/config", OrgConfigHandler)
	// JWT Middleware
	auth.Use(helper.JWTMiddleware())
	// Restricted Routes
	auth.Post("/reset-password", ResetPasswordHandler)
}

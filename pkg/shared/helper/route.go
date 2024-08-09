package helper

import (
	"github.com/gofiber/fiber/v2"
)

func CreateRouteGroup(app *fiber.App, path string, desc string) fiber.Router {
	r := app.Group(path)
	//without JWT Token validation (without auth)
	r.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(desc)
	})
	// JWT Middleware
	r.Use(JWTMiddleware())
	// r.Use(cache.New(cache.Config{
	// 	Next: func(c *fiber.Ctx) bool {
	// 		return c.Query("refresh") == "true"
	// 	},
	// 	Expiration:   30 * time.Minute,
	// 	CacheControl: true,
	// 	KeyGenerator: func(c *fiber.Ctx) string {
	// 		return c.Path() + "|" + c.Get("OrgId")
	// 	},
	// }))
	return r
}

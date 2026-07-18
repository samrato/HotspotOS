package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"hotspotos/packages/auth"
)

// JWTMiddleware validates JWT tokens for administrative routes
func JWTMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization token"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}

		claims, err := auth.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("admin", claims)
		return c.Next()
	}
}

// InternalAuthMiddleware secures calls between microservices (e.g. payment-service and api)
func InternalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("X-Internal-Token")
		if token != "internal_secret_token" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "unauthorized internal call"})
		}
		return c.Next()
	}
}

package middleware

import "github.com/gofiber/fiber/v3/middleware/cors"

func NewCORS() cors.Config {
	return cors.Config{
		AllowOrigins:     []string{"https://find-vibe.vercel.app", "http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "ngrok-skip-browser-warning"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
	}
}

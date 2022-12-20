package middleware

import "github.com/rs/cors"

// NewDefaultBkCors --
func NewDefaultBkCors() *cors.Cors {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders: []string{
			"Content-Type", "Content-Length", "Accept-Encoding",
			"X-CSRF-Token", "Authorization",
			"X-Access-Token",
		},
		AllowCredentials: true,
	})
	return c
}

package main

import (
	"fmt"
	"net/http"

	"github.com/rs/cors"
	"github.com/thatrajaryan/web-server/api"
)

func main() {
	if err := api.InitDB("postgres"); err != nil {
		api.LogError(fmt.Sprintf("Failed to initialize database: %v", err))
	}

	api.LogInfo("Starting Management API on port 8080")
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Wrap the mux with CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: true,
	})

	if err := http.ListenAndServe(":8080", c.Handler(mux)); err != nil {
		fmt.Printf("Failed to start API server: %v\n", err)
	}
}

package main

import (
	"fmt"
	"net/http"

	"github.com/thatrajaryan/web-server/api"
)

func main() {
	fmt.Println("Starting Management API on port 8080")
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Failed to start API server: %v\n", err)
	}
}

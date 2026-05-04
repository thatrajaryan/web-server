package server

import (
	"fmt"
	"net/http"
	"sync"
)

func Initialize() {
	// WaitGroup keeps the main program running as long as the servers are active
	var wg sync.WaitGroup
	
	// Define 5 distinct ports
	ports := []int{8081, 8082, 8083, 8084, 8085}

	for _, port := range ports {
		wg.Add(1)

		// Each server runs in its own Goroutine to accept concurrent connections
		go func(p int) {
			defer wg.Done()
			
			// Create a private mux so servers don't share routes
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello World from port %d", p)
			})

			addr := fmt.Sprintf(":%d", p)
			fmt.Printf("Server listening on http://localhost%s\n", addr)

			// This call blocks the Goroutine, but not the Main function
			if err := http.ListenAndServe(addr, mux); err != nil {
				fmt.Printf("Error on port %d: %v\n", p, err)
			}
		}(port)
	}

	// Main blocks here until all servers are shut down
	wg.Wait()
}
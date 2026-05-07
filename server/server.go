package server

import (
	"fmt"
	"net/http"
	"sync"
	"github.com/thatrajaryan/web-server/common"
)

type ServerBlock struct {
	IP   string
	Port int
}

func (b *ServerBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *ServerBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *ServerBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *ServerBlock) Delete() error {
	return nil
}

func (b *ServerBlock) Status() string {
	return "Active"
}

func (b *ServerBlock) Start() error {
	return nil
}

func (b *ServerBlock) Stop() error {
	return nil
}

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
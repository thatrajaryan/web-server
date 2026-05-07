package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/thatrajaryan/web-server/common"
	"github.com/thatrajaryan/web-server/api_gateway"
	"github.com/thatrajaryan/web-server/load_balancer"
	"github.com/thatrajaryan/web-server/code"
	"github.com/thatrajaryan/web-server/kafka"
	"github.com/thatrajaryan/web-server/database"
	"github.com/thatrajaryan/web-server/rate_limiter"
	"github.com/thatrajaryan/web-server/server"
	"github.com/thatrajaryan/web-server/cdn"
)

// BlockRegistry stores created blocks by their ID
var (
	registry = make(map[string]common.Block)
	regMu    sync.RWMutex
)

type CreateBlockRequest struct {
	ID     string                 `json:"id"`
	Config map[string]interface{} `json:"config"`
}

type UpdateBlockRequest struct {
	Config map[string]interface{} `json:"config"`
}

type ConnectRequest struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func RegisterRoutes(mux *http.ServeMux) {
	// Block Creation
	mux.HandleFunc("/create/api-gateway", handleCreateBlock("api_gateway"))
	mux.HandleFunc("/create/load-balancer", handleCreateBlock("load_balancer"))
	mux.HandleFunc("/create/code", handleCreateBlock("code"))
	mux.HandleFunc("/create/kafka", handleCreateBlock("kafka"))
	mux.HandleFunc("/create/database", handleCreateBlock("database"))
	mux.HandleFunc("/create/rate-limiter", handleCreateBlock("rate_limiter"))
	mux.HandleFunc("/create/server", handleCreateBlock("server"))
	mux.HandleFunc("/create/cdn", handleCreateBlock("cdn"))

	// Connection
	mux.HandleFunc("/create/connection", handleConnect)

	// Lifecycle Management
	mux.HandleFunc("/blocks", handleListBlocks)
	mux.HandleFunc("/block/update", handleUpdateBlock)
	mux.HandleFunc("/block/delete", handleDeleteBlock)
	mux.HandleFunc("/block/status", handleBlockStatus)
	mux.HandleFunc("/block/start", handleBlockStart)
	mux.HandleFunc("/block/stop", handleBlockStop)
}

func handleCreateBlock(blockType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handleOptions(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateBlockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
			return
		}

		var block common.Block
		switch blockType {
		case "api_gateway":
			block = &api_gateway.ApiGatewayBlock{}
		case "load_balancer":
			block = &load_balancer.LoadBalancerBlock{}
		case "code":
			block = &code.CodeBlock{}
		case "kafka":
			block = &kafka.KafkaBlock{}
		case "database":
			block = &database.DatabaseBlock{}
		case "rate_limiter":
			block = &rate_limiter.RateLimiterBlock{}
		case "server":
			block = &server.ServerBlock{}
		case "cdn":
			block = &cdn.CdnBlock{}
		default:
			sendResponse(w, http.StatusBadRequest, "Error", "Unknown block type", nil)
			return
		}

		if err := block.Create(req.Config); err != nil {
			sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to create block: %v", err), nil)
			return
		}

		regMu.Lock()
		registry[req.ID] = block
		regMu.Unlock()

		sendResponse(w, http.StatusCreated, "Success", fmt.Sprintf("%s created with ID %s", blockType, req.ID), nil)
	}
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
		return
	}

	regMu.RLock()
	fromBlock, fromOk := registry[req.FromID]
	toBlock, toOk := registry[req.ToID]
	regMu.RUnlock()

	if !fromOk || !toOk {
		sendResponse(w, http.StatusNotFound, "Error", "One or both blocks not found", nil)
		return
	}

	if err := fromBlock.Connect(toBlock); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to connect blocks: %v", err), nil)
		return
	}

	sendResponse(w, http.StatusOK, "Success", fmt.Sprintf("Connected %s to %s", req.FromID, req.ToID), nil)
}

func handleUpdateBlock(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if !ok {
		sendResponse(w, http.StatusNotFound, "Error", "Block not found", nil)
		return
	}

	var req UpdateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
		return
	}

	if err := block.Update(req.Config); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to update block: %v", err), nil)
		return
	}
	sendResponse(w, http.StatusOK, "Success", "Block updated", nil)
}

func handleDeleteBlock(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	regMu.Lock()
	block, ok := registry[id]
	if ok {
		block.Delete()
		delete(registry, id)
	}
	regMu.Unlock()

	if !ok {
		sendResponse(w, http.StatusNotFound, "Error", "Block not found", nil)
		return
	}
	sendResponse(w, http.StatusOK, "Success", "Block deleted", nil)
}

func handleBlockStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if !ok {
		sendResponse(w, http.StatusNotFound, "Error", "Block not found", nil)
		return
	}
	sendResponse(w, http.StatusOK, "Success", "Block status retrieved", block.Status())
}

func handleBlockStart(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if !ok {
		sendResponse(w, http.StatusNotFound, "Error", "Block not found", nil)
		return
	}
	if err := block.Start(); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to start block: %v", err), nil)
		return
	}
	sendResponse(w, http.StatusOK, "Success", "Block started", nil)
}

func handleBlockStop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if !ok {
		sendResponse(w, http.StatusNotFound, "Error", "Block not found", nil)
		return
	}
	if err := block.Stop(); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to stop block: %v", err), nil)
		return
	}
	sendResponse(w, http.StatusOK, "Success", "Block stopped", nil)
}

func handleListBlocks(w http.ResponseWriter, r *http.Request) {
	regMu.RLock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	regMu.RUnlock()
	sendResponse(w, http.StatusOK, "Success", "Blocks listed", ids)
}

func sendResponse(w http.ResponseWriter, code int, status, message string, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{Status: status, Message: message, Data: data})
}

func handleOptions(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

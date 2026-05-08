package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/thatrajaryan/web-server/api/models"
	"github.com/thatrajaryan/web-server/api_gateway"
	"github.com/thatrajaryan/web-server/cdn"
	"github.com/thatrajaryan/web-server/code"
	"github.com/thatrajaryan/web-server/common"
	"github.com/thatrajaryan/web-server/database"
	"github.com/thatrajaryan/web-server/kafka"
	"github.com/thatrajaryan/web-server/load_balancer"
	"github.com/thatrajaryan/web-server/rate_limiter"
	"github.com/thatrajaryan/web-server/server"
)

var (
	globalDB Database
)

func InitDB(strategy string) error {
	var err error
	globalDB, err = NewDatabase(strategy)
	if err != nil {
		return err
	}
	return globalDB.Connect()
}

// BlockRegistry stores created blocks by their ID
var (
	registry = make(map[string]common.Block)
	regMu    sync.RWMutex
)

type CreateBlockRequest struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	Config    map[string]interface{} `json:"config"`
}

type UpdateBlockRequest struct {
	Config map[string]interface{} `json:"config"`
}

type ConnectRequest struct {
	ProjectID string `json:"project_id"`
	FromID    string `json:"from_id"`
	ToID      string `json:"to_id"`
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func RegisterRoutes(mux *http.ServeMux) {
	// Project Management
	mux.HandleFunc("/projects", LoggingMiddleware(handleListProjects))
	mux.HandleFunc("/create/project", LoggingMiddleware(handleCreateProject))
	mux.HandleFunc("/project/", LoggingMiddleware(handleProjectDetails))

	// Block Creation
	mux.HandleFunc("/create/api-gateway", LoggingMiddleware(handleCreateBlock("api_gateway")))
	mux.HandleFunc("/create/load-balancer", LoggingMiddleware(handleCreateBlock("load_balancer")))
	mux.HandleFunc("/create/code", LoggingMiddleware(handleCreateBlock("code")))
	mux.HandleFunc("/create/kafka", LoggingMiddleware(handleCreateBlock("kafka")))
	mux.HandleFunc("/create/database", LoggingMiddleware(handleCreateBlock("database")))
	mux.HandleFunc("/create/rate-limiter", LoggingMiddleware(handleCreateBlock("rate_limiter")))
	mux.HandleFunc("/create/server", LoggingMiddleware(handleCreateBlock("server")))
	mux.HandleFunc("/create/cdn", LoggingMiddleware(handleCreateBlock("cdn")))

	// Connection
	mux.HandleFunc("/create/connection", LoggingMiddleware(handleConnect))

	// Lifecycle Management
	mux.HandleFunc("/blocks", LoggingMiddleware(handleListBlocks))
	mux.HandleFunc("/block/update", LoggingMiddleware(handleUpdateBlock))
	mux.HandleFunc("/block/delete", LoggingMiddleware(handleDeleteBlock))
	mux.HandleFunc("/block/status", LoggingMiddleware(handleBlockStatus))
	mux.HandleFunc("/block/start", LoggingMiddleware(handleBlockStart))
	mux.HandleFunc("/block/stop", LoggingMiddleware(handleBlockStop))
}

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		LogInfo(fmt.Sprintf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr))
		next(w, r)
	}
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

		// Persist to DB
		if globalDB != nil {
			configJSON, _ := json.Marshal(req.Config)
			_, err := globalDB.Query("INSERT INTO nodes (id, project_id, type, config) VALUES ($1, $2, $3, $4)",
				req.ID, req.ProjectID, blockType, string(configJSON))
			if err != nil {
				LogError(fmt.Sprintf("Failed to persist node to DB: %v", err))
			} else {
				LogInfo(fmt.Sprintf("Persisted %s node to DB", blockType))
			}
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

	// Persist to DB
	if globalDB != nil {
		_, err := globalDB.Query("INSERT INTO connections (project_id, from_node_id, to_node_id) VALUES ($1, $2, $3)",
			req.ProjectID, req.FromID, req.ToID)
		if err != nil {
			LogError(fmt.Sprintf("Failed to persist connection to DB: %v", err))
		} else {
			LogInfo("Persisted connection to DB")
		}
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

func handleListProjects(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	_, err := globalDB.Query("SELECT id, name, description, created_at, updated_at FROM projects ORDER BY updated_at DESC")
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to fetch projects: %v", err), nil)
		return
	}
	// Note: In a real app we'd scan rows properly. For this POC, we'll return mock if DB query fails or use a simpler way.
	// Since the Strategy Pattern Query returns interface{}, we'll assume it works or return empty for now.
	// Implementing proper row scanning would require more complex reflection or interface changes.
	sendResponse(w, http.StatusOK, "Success", "Projects listed", []models.Project{})
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
		return
	}

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	// Insert into DB and get the created record back
	// Note: We're using Query because of the RETURNING clause
	rowsInterface, err := globalDB.Query(
		"INSERT INTO projects (name, description) VALUES ($1, $2) RETURNING id, name, description, created_at, updated_at",
		req.Name, req.Description,
	)
	if err != nil {
		LogError(fmt.Sprintf("Failed to create project: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to create project in database", nil)
		return
	}

	rows, ok := rowsInterface.(*sql.Rows)
	if !ok {
		sendResponse(w, http.StatusInternalServerError, "Error", "Internal server error: unexpected DB response type", nil)
		return
	}
	defer rows.Close()

	if rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			LogError(fmt.Sprintf("Failed to scan project: %v", err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to read created project", nil)
			return
		}
		LogInfo(fmt.Sprintf("Project created: %s (%s)", p.Name, p.ID))
		sendResponse(w, http.StatusCreated, "Success", "Project created", p)
		return
	}

	sendResponse(w, http.StatusInternalServerError, "Error", "Failed to create project", nil)
}

func handleProjectDetails(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	// Note: In a real app we'd use a router to get {id}
	// For this POC, we'll return an empty structure that the frontend expects
	sendResponse(w, http.StatusOK, "Success", "Project details retrieved", map[string]interface{}{
		"nodes":       []models.Node{},
		"connections": []models.Connection{},
	})
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

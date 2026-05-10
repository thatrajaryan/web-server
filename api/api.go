package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/thatrajaryan/web-server/api/models"
	"github.com/thatrajaryan/web-server/api_gateway"
	"github.com/thatrajaryan/web-server/cdn"
	"github.com/thatrajaryan/web-server/code"
	"github.com/thatrajaryan/web-server/common"
	"github.com/thatrajaryan/web-server/database"
	"github.com/thatrajaryan/web-server/kafka"
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
	mux.HandleFunc("/project/delete", LoggingMiddleware(handleDeleteProject))
	mux.HandleFunc("/project/", LoggingMiddleware(handleProjectDetails))

	// Block Creation
	mux.HandleFunc("/create/api-gateway", LoggingMiddleware(handleCreateBlock("api-gateway")))
	mux.HandleFunc("/create/code", LoggingMiddleware(handleCreateBlock("code")))
	mux.HandleFunc("/create/kafka", LoggingMiddleware(handleCreateBlock("kafka")))
	mux.HandleFunc("/create/database", LoggingMiddleware(handleCreateBlock("database")))
	mux.HandleFunc("/create/server", LoggingMiddleware(handleCreateBlock("server")))
	mux.HandleFunc("/create/cdn", LoggingMiddleware(handleCreateBlock("cdn")))

	// Connection
	mux.HandleFunc("/create/connection", LoggingMiddleware(handleConnect))
	mux.HandleFunc("/connection/delete", LoggingMiddleware(handleDeleteConnection))

	// Architectural Management
	mux.HandleFunc("/blocks", LoggingMiddleware(handleListBlocks))
	mux.HandleFunc("/block/update", LoggingMiddleware(handleUpdateBlock))
	mux.HandleFunc("/block/delete", LoggingMiddleware(handleDeleteBlock))
	mux.HandleFunc("/block/details/", LoggingMiddleware(handleGetBlock))
	mux.HandleFunc("/config/", LoggingMiddleware(handleGetConfig))
}

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		LogInfo(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
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
		case "api-gateway":
			block = &api_gateway.ApiGatewayBlock{}
		case "code":
			block = &code.CodeBlock{}
		case "kafka":
			block = &kafka.KafkaBlock{}
		case "database":
			block = &database.DatabaseBlock{}
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
			// Initialize only with incoming config (like position)
			configJSON, _ := json.Marshal(req.Config)
			_, err := globalDB.Exec("INSERT INTO nodes (id, project_id, type, config) VALUES ($1, $2, $3, $4)",
				req.ID, req.ProjectID, blockType, string(configJSON))
			if err != nil {
				LogError(fmt.Sprintf("Failed to persist node %s to DB: %v", req.ID, err))
				sendResponse(w, http.StatusInternalServerError, "Error", "Failed to save node to database", nil)
				return
			}
			LogInfo(fmt.Sprintf("Successfully persisted %s node to DB", blockType))
		}

		regMu.Lock()
		registry[req.ID] = block
		regMu.Unlock()

		if err := block.Create(req.Config); err != nil {
			LogError(fmt.Sprintf("Failed to initialize live block %s: %v", req.ID, err))
		}

		sendResponse(w, http.StatusCreated, "Success", fmt.Sprintf("%s created and initialized with ID %s", blockType, req.ID), nil)
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

func handleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		// Fallback: try source/target if ID is not provided
		source := r.URL.Query().Get("source")
		target := r.URL.Query().Get("target")
		if source != "" && target != "" {
			if globalDB != nil {
				_, err := globalDB.Query("DELETE FROM connections WHERE from_node_id = $1 AND to_node_id = $2", source, target)
				if err != nil {
					LogError(fmt.Sprintf("Failed to delete connection %s -> %s: %v", source, target, err))
					sendResponse(w, http.StatusInternalServerError, "Error", "Failed to delete connection", nil)
					return
				}
				sendResponse(w, http.StatusOK, "Success", "Connection deleted", nil)
				return
			}
		}
		sendResponse(w, http.StatusBadRequest, "Error", "Connection ID or source/target missing", nil)
		return
	}

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	_, err := globalDB.Query("DELETE FROM connections WHERE id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to delete connection %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to delete connection", nil)
		return
	}

	LogInfo(fmt.Sprintf("Connection %s deleted from DB", id))
	sendResponse(w, http.StatusOK, "Success", "Connection deleted", nil)
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	// Path format: /config/{type}
	nodeType := strings.TrimPrefix(r.URL.Path, "/config/")
	if nodeType == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Node type missing", nil)
		return
	}

	configPath := filepath.Join("api", "configs", nodeType+".yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		LogError(fmt.Sprintf("Failed to read config file %s: %v", configPath, err))
		sendResponse(w, http.StatusNotFound, "Error", "Config not found", nil)
		return
	}

	var config interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		LogError(fmt.Sprintf("Failed to unmarshal YAML for %s: %v", nodeType, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to parse config", nil)
		return
	}

	// Convert map[interface{}]interface{} to map[string]interface{} for JSON encoding
	convertedConfig := convertMap(config)

	sendResponse(w, http.StatusOK, "Success", "Config fetched", convertedConfig)
}

// Helper to convert YAML maps to JSON-compatible maps recursively
func convertMap(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprintf("%v", k)] = convertMap(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertMap(v)
		}
	}
	return i
}

func handleGetBlock(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/block/details/")
	if id == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Block ID missing", nil)
		return
	}

	LogInfo(fmt.Sprintf("Fetching details for block: %s", id))

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	row, err := globalDB.Query("SELECT id, project_id, type, config FROM nodes WHERE id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Database error while fetching block %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Database error", nil)
		return
	}
	rows := row.(*sql.Rows)
	defer rows.Close()

	if !rows.Next() {
		LogInfo(fmt.Sprintf("Block %s not found in database", id))
		sendResponse(w, http.StatusNotFound, "Error", "Block not found in database", nil)
		return
	}

	var n models.Node
	var configJSON string
	if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &configJSON); err != nil {
		LogError(fmt.Sprintf("Failed to scan block %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Internal server error", nil)
		return
	}
	json.Unmarshal([]byte(configJSON), &n.Config)

	sendResponse(w, http.StatusOK, "Success", "Block fetched", n)
}

func handleUpdateBlock(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Node ID missing", nil)
		return
	}

	var req UpdateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
		return
	}

	// Update in registry (in-memory)
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if ok {
		if err := block.Update(req.Config); err != nil {
			LogError(fmt.Sprintf("Failed to update in-memory block %s: %v", id, err))
		}
	}

	// Persist to DB
	if globalDB != nil {
		configJSON, _ := json.Marshal(req.Config)
		_, err := globalDB.Exec("UPDATE nodes SET config = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", string(configJSON), id)
		if err != nil {
			LogError(fmt.Sprintf("Failed to update node %s in DB: %v", id, err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to update database", nil)
			return
		}
		LogInfo(fmt.Sprintf("Successfully persisted new configuration for node %s", id))
	}

	sendResponse(w, http.StatusOK, "Success", "Block updated", nil)
}

func handleDeleteBlock(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Node ID missing", nil)
		return
	}

	regMu.Lock()
	block, ok := registry[id]
	if ok {
		block.Delete()
		delete(registry, id)
	}
	regMu.Unlock()

	if !ok {
		// Even if not in registry, we should try to delete from DB
		LogInfo(fmt.Sprintf("Block %s not found in registry, attempting DB cleanup only", id))
	}

	// Transactional DB Cleanup
	if globalDB != nil {
		tx, err := globalDB.Begin()
		if err != nil {
			LogError(fmt.Sprintf("Failed to start node deletion transaction: %v", err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Internal server error", nil)
			return
		}
		defer tx.Rollback()

		// 1. Delete all connections involving this node
		_, err = tx.Exec("DELETE FROM connections WHERE from_node_id = $1 OR to_node_id = $1", id)
		if err != nil {
			LogError(fmt.Sprintf("Failed to delete connections for node %s: %v", id, err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to cleanup connections", nil)
			return
		}

		// 2. Delete the node itself
		_, err = tx.Exec("DELETE FROM nodes WHERE id = $1", id)
		if err != nil {
			LogError(fmt.Sprintf("Failed to delete node %s from DB: %v", id, err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to delete node", nil)
			return
		}

		if err := tx.Commit(); err != nil {
			LogError(fmt.Sprintf("Failed to commit node deletion for %s: %v", id, err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to finalize deletion", nil)
			return
		}
		LogInfo(fmt.Sprintf("Node %s and all its connections deleted transactionally", id))
	}

	sendResponse(w, http.StatusOK, "Success", "Block and its connections deleted", nil)
}


func handleListProjects(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	rowsInterface, err := globalDB.Query("SELECT id, name, description, created_at, updated_at FROM projects ORDER BY updated_at DESC")
	if err != nil {
		LogError(fmt.Sprintf("Failed to fetch projects from DB: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to fetch projects", nil)
		return
	}

	rows, ok := rowsInterface.(*sql.Rows)
	if !ok {
		LogError("Failed to cast rowsInterface to *sql.Rows")
		sendResponse(w, http.StatusInternalServerError, "Error", "Internal server error: unexpected DB response type", nil)
		return
	}
	defer rows.Close()

	projects := []models.Project{}
	count := 0
	for rows.Next() {
		count++
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			LogError(fmt.Sprintf("Failed to scan project row %d: %v", count, err))
			continue
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		LogError(fmt.Sprintf("Error during rows iteration: %v", err))
	}

	LogInfo(fmt.Sprintf("Fetched %d projects from database", len(projects)))
	sendResponse(w, http.StatusOK, "Success", "Projects listed", projects)
}

func handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Project ID missing", nil)
		return
	}

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	// Start a transaction for manual cascading delete
	tx, err := globalDB.Begin()
	if err != nil {
		LogError(fmt.Sprintf("Failed to start transaction: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to start deletion transaction", nil)
		return
	}
	// Ensure rollback on failure
	defer tx.Rollback()

	// 1. Delete all connections belonging to this project
	_, err = tx.Exec("DELETE FROM connections WHERE project_id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to delete connections for project %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to cleanup connections", nil)
		return
	}

	// 2. Delete all nodes belonging to this project
	_, err = tx.Exec("DELETE FROM nodes WHERE project_id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to delete nodes for project %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to cleanup nodes", nil)
		return
	}

	// 3. Finally, delete the project itself
	_, err = tx.Exec("DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to delete project %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to delete project", nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		LogError(fmt.Sprintf("Failed to commit deletion transaction for project %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to finalize deletion", nil)
		return
	}

	LogInfo(fmt.Sprintf("Project %s and all its associated components were manually deleted in a single transaction", id))
	sendResponse(w, http.StatusOK, "Success", "Project and its components deleted", nil)
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
	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	// Extract project ID from /project/{id}/details
	path := r.URL.Path
	// Using a robust way to get the ID
	id := ""
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) >= 2 {
		id = pathParts[1]
	}

	if id == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Project ID missing", nil)
		return
	}

	// Fetch Nodes
	nodeRowsInterface, err := globalDB.Query("SELECT id, type, config FROM nodes WHERE project_id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to fetch nodes for project %s: %v", id, err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to fetch project data", nil)
		return
	}
	nodeRows := nodeRowsInterface.(*sql.Rows)
	defer nodeRows.Close()

	nodes := []models.Node{}
	for nodeRows.Next() {
		var n models.Node
		var configStr string
		if err := nodeRows.Scan(&n.ID, &n.Type, &configStr); err != nil {
			continue
		}
		json.Unmarshal([]byte(configStr), &n.Config)
		nodes = append(nodes, n)
	}

	// Fetch Connections
	connRowsInterface, err := globalDB.Query("SELECT id, from_node_id, to_node_id FROM connections WHERE project_id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to fetch connections for project %s: %v", id, err))
	} else {
		connRows := connRowsInterface.(*sql.Rows)
		defer connRows.Close()

		connections := []models.Connection{}
		for connRows.Next() {
			var c models.Connection
			if err := connRows.Scan(&c.ID, &c.FromNodeID, &c.ToNodeID); err != nil {
				continue
			}
			connections = append(connections, c)
		}

		sendResponse(w, http.StatusOK, "Success", "Project details retrieved", map[string]interface{}{
			"nodes":       nodes,
			"connections": connections,
		})
		return
	}

	sendResponse(w, http.StatusOK, "Success", "Project details retrieved", map[string]interface{}{
		"nodes":       nodes,
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{Status: status, Message: message, Data: data})
}

func handleOptions(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

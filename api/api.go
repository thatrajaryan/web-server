package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/thatrajaryan/web-server/api/models"
	"github.com/thatrajaryan/web-server/ai"
	"github.com/thatrajaryan/web-server/api_gateway"
	"github.com/thatrajaryan/web-server/cdn"
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
	HookCode  string `json:"hook_code"`
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SaveProjectRequest struct {
	ProjectID   string              `json:"project_id"`
	Nodes       []models.Node       `json:"nodes"`
	Connections []models.Connection `json:"connections"`
}

type YamlProjectConfig struct {
	ProjectName string `yaml:"project_name"`
	Nodes       []struct {
		ID     string                 `yaml:"id"`
		Type   string                 `yaml:"type"`
		Config map[string]interface{} `yaml:"config"`
	} `yaml:"nodes"`
	Connections []struct {
		From     string `yaml:"from"`
		To       string `yaml:"to"`
		HookCode string `yaml:"hook_code"`
	} `yaml:"connections"`
}

func RegisterRoutes(mux *http.ServeMux) {
	// Project Management
	mux.HandleFunc("/projects", LoggingMiddleware(handleListProjects))
	mux.HandleFunc("/create/project", LoggingMiddleware(handleCreateProject))
	mux.HandleFunc("/project/delete", LoggingMiddleware(handleDeleteProject))
	mux.HandleFunc("/project/save", LoggingMiddleware(handleSaveProject))
	mux.HandleFunc("/project/upload-config", LoggingMiddleware(handleUploadConfig))
	mux.HandleFunc("/project/", LoggingMiddleware(handleProjectDetails))

	// Block Creation
	mux.HandleFunc("/create/api-gateway", LoggingMiddleware(handleCreateBlock("api-gateway")))
	mux.HandleFunc("/create/ai", LoggingMiddleware(handleCreateBlock("ai")))
	mux.HandleFunc("/create/kafka", LoggingMiddleware(handleCreateBlock("kafka")))
	mux.HandleFunc("/create/database", LoggingMiddleware(handleCreateBlock("database")))
	mux.HandleFunc("/create/server", LoggingMiddleware(handleCreateBlock("server")))
	mux.HandleFunc("/create/cdn", LoggingMiddleware(handleCreateBlock("cdn")))
	mux.HandleFunc("/create/load-balancer", LoggingMiddleware(handleCreateBlock("load-balancer")))
	mux.HandleFunc("/create/rate-limiter", LoggingMiddleware(handleCreateBlock("rate-limiter")))

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

		block := createBlockInstance(blockType)
		if block == nil {
			sendResponse(w, http.StatusBadRequest, "Error", "Unknown block type", nil)
			return
		}

		if err := block.Create(req.Config); err != nil {
			sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to initialize block: %v", err), nil)
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

	if req.ProjectID == "" {
		sendResponse(w, http.StatusBadRequest, "Error", "Project ID is required", nil)
		return
	}

	fromBlock, err := getOrLoadBlock(req.FromID)
	if err != nil {
		sendResponse(w, http.StatusNotFound, "Error", fmt.Sprintf("Source block %s not found: %v", req.FromID, err), nil)
		return
	}

	toBlock, err := getOrLoadBlock(req.ToID)
	if err != nil {
		sendResponse(w, http.StatusNotFound, "Error", fmt.Sprintf("Target block %s not found: %v", req.ToID, err), nil)
		return
	}

	if err := fromBlock.Connect(toBlock); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to connect blocks: %v", err), nil)
		return
	}

	// Persist to DB
	if globalDB != nil {
		_, err := globalDB.Exec("INSERT INTO connections (project_id, from_node_id, to_node_id, hook_code) VALUES ($1, $2, $3, $4)",
			req.ProjectID, req.FromID, req.ToID, req.HookCode)
		if err != nil {
			LogError(fmt.Sprintf("Failed to persist connection to DB: %v", err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to save connection to database", nil)
			return
		}
		LogInfo("Persisted connection to DB")
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
				_, err := globalDB.Exec("DELETE FROM connections WHERE from_node_id = $1 AND to_node_id = $2", source, target)
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

	_, err := globalDB.Exec("DELETE FROM connections WHERE id = $1", id)
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
	block, err := getOrLoadBlock(id)
	if err == nil {
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

	block, err := getOrLoadBlock(id)
	if err == nil {
		block.Delete()
	}
	regMu.Lock()
	delete(registry, id)
	regMu.Unlock()

	if err != nil {
		// Even if not in registry/DB as a live block, we should try to delete from DB
		LogInfo(fmt.Sprintf("Block %s not found for live deletion, attempting DB cleanup only", id))
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
	nodeRowsInterface, err := globalDB.Query("SELECT id, project_id, type, config, created_at, updated_at FROM nodes WHERE project_id = $1", id)
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
		if err := nodeRows.Scan(&n.ID, &n.ProjectID, &n.Type, &configStr, &n.CreatedAt, &n.UpdatedAt); err != nil {
			LogError(fmt.Sprintf("Failed to scan node row for project %s: %v", id, err))
			continue
		}
		if err := json.Unmarshal([]byte(configStr), &n.Config); err != nil {
			LogError(fmt.Sprintf("Failed to unmarshal config for node %s: %v", n.ID, err))
		}
		nodes = append(nodes, n)
	}

	// Fetch Connections
	connRowsInterface, err := globalDB.Query("SELECT id, project_id, from_node_id, to_node_id, hook_code, created_at FROM connections WHERE project_id = $1", id)
	if err != nil {
		LogError(fmt.Sprintf("Failed to fetch connections for project %s: %v", id, err))
	} else {
		connRows := connRowsInterface.(*sql.Rows)
		defer connRows.Close()

		connections := []models.Connection{}
		for connRows.Next() {
			var c models.Connection
			if err := connRows.Scan(&c.ID, &c.ProjectID, &c.FromNodeID, &c.ToNodeID, &c.HookCode, &c.CreatedAt); err != nil {
				LogError(fmt.Sprintf("Failed to scan connection row for project %s: %v", id, err))
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

func handleSaveProject(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SaveProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Invalid request body", nil)
		return
	}

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	tx, err := globalDB.Begin()
	if err != nil {
		LogError(fmt.Sprintf("Failed to start transaction: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to start save transaction", nil)
		return
	}
	defer tx.Rollback()

	// 1. Upsert Nodes
	for _, n := range req.Nodes {
		configJSON, _ := json.Marshal(n.Config)
		_, err := tx.Exec(`
			INSERT INTO nodes (id, project_id, type, config) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (id) 
			DO UPDATE SET config = EXCLUDED.config, updated_at = CURRENT_TIMESTAMP`,
			n.ID, req.ProjectID, n.Type, string(configJSON))
		if err != nil {
			LogError(fmt.Sprintf("Failed to upsert node %s: %v", n.ID, err))
			sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to save node %s", n.ID), nil)
			return
		}

		// Also ensure live registry is updated
		if _, err := getOrLoadBlock(n.ID); err != nil {
			LogError(fmt.Sprintf("Failed to load/init live block %s: %v", n.ID, err))
		}
	}

	// 2. Upsert Connections
	for _, c := range req.Connections {
		_, err := tx.Exec(`
			INSERT INTO connections (project_id, from_node_id, to_node_id, hook_code) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (from_node_id, to_node_id) 
			DO UPDATE SET hook_code = EXCLUDED.hook_code`,
			req.ProjectID, c.FromNodeID, c.ToNodeID, c.HookCode)
		if err != nil {
			LogError(fmt.Sprintf("Failed to upsert connection: %v", err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to save connections", nil)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		LogError(fmt.Sprintf("Failed to commit save transaction: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to finalize save", nil)
		return
	}

	LogInfo(fmt.Sprintf("Project %s saved successfully with %d nodes and %d connections", req.ProjectID, len(req.Nodes), len(req.Connections)))
	sendResponse(w, http.StatusOK, "Success", "Project saved successfully", nil)
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

func createBlockInstance(blockType string) common.Block {
	switch blockType {
	case "api-gateway":
		return &api_gateway.ApiGatewayBlock{}
	case "ai":
		return &ai.AIBlock{}
	case "kafka":
		return &kafka.KafkaBlock{}
	case "database":
		return &database.DatabaseBlock{}
	case "server":
		return &server.ServerBlock{}
	case "cdn":
		return &cdn.CdnBlock{}
	case "load-balancer":
		return &load_balancer.LoadBalancerBlock{}
	case "rate-limiter":
		return &rate_limiter.RateLimiterBlock{}
	default:
		return nil
	}
}

func getOrLoadBlock(id string) (common.Block, error) {
	regMu.RLock()
	block, ok := registry[id]
	regMu.RUnlock()

	if ok {
		return block, nil
	}

	// Not in registry, try to load from DB
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rowInterface, err := globalDB.Query("SELECT type, config FROM nodes WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	rows := rowInterface.(*sql.Rows)
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("node not found in database")
	}

	var blockType, configJSON string
	if err := rows.Scan(&blockType, &configJSON); err != nil {
		return nil, err
	}

	var config map[string]interface{}
	json.Unmarshal([]byte(configJSON), &config)

	block = createBlockInstance(blockType)
	if block == nil {
		return nil, fmt.Errorf("unknown block type: %s", blockType)
	}

	if err := block.Create(config); err != nil {
		LogError(fmt.Sprintf("Failed to initialize loaded block %s: %v", id, err))
		// Continue anyway as we have the instance
	}

	regMu.Lock()
	registry[id] = block
	regMu.Unlock()

	LogInfo(fmt.Sprintf("Successfully loaded block %s from DB into registry", id))
	return block, nil
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

func handleUploadConfig(w http.ResponseWriter, r *http.Request) {
	if handleOptions(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		sendResponse(w, http.StatusBadRequest, "Error", "Failed to parse form", nil)
		return
	}

	file, _, err := r.FormFile("config")
	if err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", "Config file missing", nil)
		return
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to read file", nil)
		return
	}

	// 2. Parse YAML
	var config YamlProjectConfig
	if err := yaml.Unmarshal(fileBytes, &config); err != nil {
		sendResponse(w, http.StatusBadRequest, "Error", fmt.Sprintf("Invalid YAML: %v", err), nil)
		return
	}

	if globalDB == nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Database not initialized", nil)
		return
	}

	// 3. Database Transaction
	tx, err := globalDB.Begin()
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to start transaction", nil)
		return
	}
	defer tx.Rollback()

	// 3a. Create Project
	var projectID string
	err = tx.QueryRow(
		"INSERT INTO projects (name, description) VALUES ($1, $2) RETURNING id",
		config.ProjectName, "Imported from YAML config",
	).Scan(&projectID)
	if err != nil {
		LogError(fmt.Sprintf("Failed to create project from YAML: %v", err))
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to create project", nil)
		return
	}

	// 3b. Insert Nodes
	for _, n := range config.Nodes {
		configJSON, _ := json.Marshal(n.Config)
		_, err := tx.Exec(
			"INSERT INTO nodes (id, project_id, type, config) VALUES ($1, $2, $3, $4)",
			n.ID, projectID, n.Type, string(configJSON),
		)
		if err != nil {
			LogError(fmt.Sprintf("Failed to insert node %s: %v", n.ID, err))
			sendResponse(w, http.StatusInternalServerError, "Error", fmt.Sprintf("Failed to insert node %s", n.ID), nil)
			return
		}
	}

	// 3c. Insert Connections
	for _, c := range config.Connections {
		_, err := tx.Exec(
			"INSERT INTO connections (project_id, from_node_id, to_node_id, hook_code) VALUES ($1, $2, $3, $4)",
			projectID, c.From, c.To, c.HookCode,
		)
		if err != nil {
			LogError(fmt.Sprintf("Failed to insert connection: %v", err))
			sendResponse(w, http.StatusInternalServerError, "Error", "Failed to insert connections", nil)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		sendResponse(w, http.StatusInternalServerError, "Error", "Failed to commit transaction", nil)
		return
	}

	LogInfo(fmt.Sprintf("Successfully imported project %s from YAML with ID %s", config.ProjectName, projectID))
	sendResponse(w, http.StatusOK, "Success", "Project imported successfully", map[string]string{"project_id": projectID})
}
